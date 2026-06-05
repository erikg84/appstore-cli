package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/dallaslabs/appctl/server/graphql"
	"github.com/dallaslabs/appctl/server/rest"
)

func main() {
	port := flag.String("port", "8080", "Port to listen on")
	uiDir := flag.String("ui-dir", "", "Directory of the built webapp to serve at / (optional)")
	flag.Parse()

	r := chi.NewRouter()
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)

	rest.Mount(r)
	graphql.Mount(r)

	if *uiDir != "" {
		if _, err := os.Stat(*uiDir); err != nil {
			log.Fatalf("ui-dir %q not found: %v", *uiDir, err)
		}
		r.Handle("/*", http.FileServer(http.Dir(*uiDir)))
		log.Printf("  UI:      http://localhost:%s/ (serving %s)", *port, *uiDir)
	}

	log.Printf("appctl server listening on :%s", *port)
	log.Printf("  REST:    http://localhost:%s/api/v1/", *port)
	log.Printf("  GraphQL: http://localhost:%s/graphql", *port)
	log.Fatal(http.ListenAndServe(":"+*port, r))
}
