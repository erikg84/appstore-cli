package rest

import "github.com/go-chi/chi/v5"

func Mount(r chi.Router) {
	h := Handler{}

	r.With(JSONContentType, CORS).Get("/health", h.Health)
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(JSONContentType)
		r.Use(CORS)
		r.Use(RequestLogger)

		r.Get("/apps", h.Apps)
		r.Get("/apps/{alias}/versions", h.Versions)
		r.Get("/apps/{alias}/builds", h.Builds)
		r.Get("/apps/{alias}/tracks", h.Tracks)
		r.Get("/apps/{alias}/reviews", h.Reviews)
		r.Get("/apps/{alias}/installs", h.Installs)
		r.Get("/apps/{alias}/iap", h.IAP)
		r.Get("/apps/{alias}/subscriptions", h.Subscriptions)
		r.Get("/apps/{alias}/testflight/groups", h.TestFlightGroups)
		r.Get("/apps/{alias}/testflight/testers", h.TestFlightTesters)
		r.Get("/reports/sales", h.SalesReport)
		r.Get("/reports/play/files", h.PlayReportFiles)
		r.Get("/reports/play/earnings", h.PlayEarnings)
		r.Get("/reports/play/sales", h.PlaySales)
		r.Get("/reports/play/installs", h.PlayInstalls)
		r.Get("/reports/play/crashes", h.PlayCrashes)
		r.Get("/reports/play/acquisition", h.PlayAcquisition)
		r.Get("/users", h.Users)
	})
}
