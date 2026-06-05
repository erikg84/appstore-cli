package graphql

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	gql "github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/go-chi/chi/v5"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/dallaslabs/appctl/core/asc"
	"github.com/dallaslabs/appctl/core/config"
	"github.com/dallaslabs/appctl/core/play"
)

// Compile-time check so the imports stay if only used in methods below.
var _ *asc.Client
var _ *play.Client

//go:embed schema.graphql
var schemaSDL string

type Resolver struct{}

// NewResolver returns a resolver with lazily-initialized clients.
// Clients are created on first use to avoid blocking startup on network I/O
// (the Google API client library probes the GCE metadata server at init time).
func NewResolver() *Resolver {
	return &Resolver{}
}

func (r *Resolver) ascClient() *asc.Client {
	creds := config.Load()
	return asc.NewClient(creds.ASCKeyID, creds.ASCIssuerID, creds.ASCKeyFile)
}

func (r *Resolver) playClient() (*play.Client, error) {
	return play.NewClient(config.Load().PlayKeyFile)
}

func Mount(r chi.Router) {
	schema := gqlparser.MustLoadSchema(&ast.Source{Name: "schema.graphql", Input: schemaSDL})
	server := handler.NewDefaultServer(&executableSchema{schema: schema, resolver: NewResolver()})
	r.Handle("/graphql", server)
}

type executableSchema struct {
	schema   *ast.Schema
	resolver *Resolver
}

func (e *executableSchema) Schema() *ast.Schema {
	return e.schema
}

func (e *executableSchema) Complexity(context.Context, string, string, int, map[string]any) (int, bool) {
	return 0, false
}

func (e *executableSchema) Exec(ctx context.Context) gql.ResponseHandler {
	opCtx := gql.GetOperationContext(ctx)
	if opCtx == nil || opCtx.Operation == nil {
		return gql.OneShot(&gql.Response{Errors: gqlerror.List{{Message: "missing operation context"}}})
	}
	if opCtx.Operation.Operation != ast.Query {
		return gql.OneShot(&gql.Response{Errors: gqlerror.List{{Message: "only query operations are supported"}}})
	}

	payload := map[string]any{}
	for _, selection := range opCtx.Operation.SelectionSet {
		field, ok := selection.(*ast.Field)
		if !ok {
			continue
		}
		key := field.Name
		if field.Alias != "" {
			key = field.Alias
		}
		value, err := e.resolveField(ctx, opCtx, field)
		if err != nil {
			return gql.OneShot(&gql.Response{Errors: gqlerror.List{{Message: err.Error()}}})
		}
		payload[key] = projectValue(value, field.SelectionSet)
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return gql.OneShot(&gql.Response{Errors: gqlerror.List{{Message: err.Error()}}})
	}
	return gql.OneShot(&gql.Response{Data: data})
}

func (e *executableSchema) resolveField(ctx context.Context, opCtx *gql.OperationContext, field *ast.Field) (any, error) {
	switch field.Name {
	case "apps":
		return e.resolver.Apps(ctx)
	case "app":
		return e.resolver.App(ctx, stringArg(opCtx, field, "alias"))
	case "versions":
		return e.resolver.Versions(ctx, stringArg(opCtx, field, "alias"))
	case "builds":
		return e.resolver.Builds(ctx, stringArg(opCtx, field, "alias"))
	case "tracks":
		return e.resolver.Tracks(ctx, stringArg(opCtx, field, "alias"))
	case "reviews":
		return e.resolver.Reviews(ctx, stringArg(opCtx, field, "alias"), storeArg(opCtx, field))
	case "iap":
		return e.resolver.IAP(ctx, stringArg(opCtx, field, "alias"), storeArg(opCtx, field))
	case "subscriptions":
		return e.resolver.Subscriptions(ctx, stringArg(opCtx, field, "alias"), storeArg(opCtx, field))
	case "testflightGroups":
		return e.resolver.TestFlightGroups(ctx, stringArg(opCtx, field, "alias"))
	case "testflightTesters":
		return e.resolver.TestFlightTesters(ctx, stringArg(opCtx, field, "alias"))
	case "users":
		return e.resolver.Users(ctx)
	default:
		return nil, fmt.Errorf("unsupported query field %q", field.Name)
	}
}

func stringArg(opCtx *gql.OperationContext, field *ast.Field, name string) string {
	arg := field.Arguments.ForName(name)
	if arg == nil {
		return ""
	}
	value, err := arg.Value.Value(opCtx.Variables)
	if err != nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	return fmt.Sprint(value)
}

func storeArg(opCtx *gql.OperationContext, field *ast.Field) string {
	value := strings.ToLower(stringArg(opCtx, field, "store"))
	if value == "" {
		return "both"
	}
	return value
}

func projectValue(value any, selectionSet ast.SelectionSet) any {
	if value == nil {
		return nil
	}

	rv := reflect.ValueOf(value)
	for rv.IsValid() && rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	if !rv.IsValid() {
		return nil
	}

	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		items := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			items[i] = projectValue(rv.Index(i).Interface(), selectionSet)
		}
		return items
	case reflect.Struct:
		if len(selectionSet) == 0 {
			return rv.Interface()
		}
		out := map[string]any{}
		for _, selection := range selectionSet {
			field, ok := selection.(*ast.Field)
			if !ok {
				continue
			}
			key := field.Name
			if field.Alias != "" {
				key = field.Alias
			}
			fieldValue, ok := extractFieldValue(rv, field.Name)
			if !ok {
				out[key] = nil
				continue
			}
			out[key] = projectValue(fieldValue.Interface(), field.SelectionSet)
		}
		return out
	default:
		return rv.Interface()
	}
}

func extractFieldValue(rv reflect.Value, name string) (reflect.Value, bool) {
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		sf := rt.Field(i)
		if sf.PkgPath != "" {
			continue
		}
		jsonName := sf.Tag.Get("json")
		if idx := strings.Index(jsonName, ","); idx >= 0 {
			jsonName = jsonName[:idx]
		}
		if jsonName == "" {
			jsonName = lowerFirst(sf.Name)
		}
		if jsonName == name || lowerFirst(sf.Name) == name {
			return rv.Field(i), true
		}
	}
	return reflect.Value{}, false
}

func lowerFirst(value string) string {
	if value == "" {
		return value
	}
	return strings.ToLower(value[:1]) + value[1:]
}
