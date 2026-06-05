package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/dallaslabs/appctl/core/asc"
	"github.com/dallaslabs/appctl/core/config"
	"github.com/dallaslabs/appctl/core/play"
	"github.com/dallaslabs/appctl/core/store"
)

type apiError struct {
	Error string `json:"error"`
}

func ascClient() *asc.Client {
	creds := config.Load()
	return asc.NewClient(creds.ASCKeyID, creds.ASCIssuerID, creds.ASCKeyFile)
}

func ascAnalyticsClient() *asc.Client {
	creds := config.Load()
	return asc.NewClient(creds.ASCAnalyticsKeyID, creds.ASCAnalyticsIssuerID, creds.ASCAnalyticsKeyFile)
}

func playClient() (*play.Client, error) {
	creds := config.Load()
	return play.NewClient(creds.PlayKeyFile)
}

func die(err error) error {
	return err
}

func appByAlias(alias string) (config.App, error) {
	app, ok := config.Apps[alias]
	if !ok {
		return config.App{}, fmt.Errorf("unknown app %q", alias)
	}
	return app, nil
}

func appSummaries() []store.AppSummary {
	aliases := make([]string, 0, len(config.Apps))
	for alias := range config.Apps {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)

	apps := make([]store.AppSummary, 0, len(aliases))
	for _, alias := range aliases {
		app := config.Apps[alias]
		apps = append(apps, store.AppSummary{
			Alias:       alias,
			Name:        app.Name,
			ASCID:       app.ASCAppID,
			PlayPackage: app.PlayPackage,
		})
	}
	return apps
}

func normalizeStore(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		value = "both"
	}
	switch value {
	case "ios", "android", "both":
		return value, nil
	default:
		return "", fmt.Errorf("invalid store %q (expected ios, android, or both)", value)
	}
}

func wantsIOS(storeName string) bool {
	return storeName == "ios" || storeName == "both"
}

func wantsAndroid(storeName string) bool {
	return storeName == "android" || storeName == "both"
}

// envelope is the standard JSON response wrapper used when --output json.
// All JSON output goes through this so the AppCtlAdapter can parse it uniformly.
type envelope struct {
	SchemaVersion string `json:"schema_version"`
	Verdict       string `json:"verdict"` // "pass" or "fail"
	Result        any    `json:"result,omitempty"`
	Error         *cliError `json:"error,omitempty"`
}

type cliError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

func render(value any, table func()) error {
	if outputFormat == "json" {
		return printJSON(value)
	}
	table()
	return nil
}

func printJSON(value any) error {
	env := envelope{
		SchemaVersion: "1",
		Verdict:       "pass",
		Result:        value,
	}
	data, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

// failJSON writes a structured error envelope to stderr and returns a non-nil error.
func failJSON(code, message, suggestion string) error {
	if outputFormat == "json" {
		env := envelope{
			SchemaVersion: "1",
			Verdict:       "fail",
			Error: &cliError{
				Code:       code,
				Message:    message,
				Suggestion: suggestion,
			},
		}
		data, _ := json.MarshalIndent(env, "", "  ")
		fmt.Println(string(data))
	}
	return fmt.Errorf("%s: %s", code, message)
}

func fetchServerJSON(path string, query url.Values, target any) error {
	base := strings.TrimRight(serverAddr, "/")
	endpoint := base + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	resp, err := http.Get(endpoint)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		var apiErr apiError
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err == nil && apiErr.Error != "" {
			return errors.New(apiErr.Error)
		}
		return fmt.Errorf("server returned %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

func trim(value string, n int) string {
	value = strings.TrimSpace(value)
	if len(value) <= n {
		return value
	}
	if n <= 3 {
		return value[:n]
	}
	return value[:n-3] + "..."
}

func listPlayUsers() ([]store.AppUser, error) {
	if strings.TrimSpace(os.Getenv("PLAY_DEVELOPER_ACCOUNT")) == "" {
		return nil, nil
	}
	client, err := playClient()
	if err != nil {
		return nil, err
	}
	return client.ListUsers("")
}
