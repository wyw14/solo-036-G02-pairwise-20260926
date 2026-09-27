package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/wyw14/solo-036/internal/service"
	"net/http"
)

func NewRouter(runtime *service.Runtime) http.Handler {
	router := chi.NewRouter()
	h := NewHandler(runtime)
	router.Get("/healthz", h.Health)
	router.Route("/api", func(r chi.Router) {
		r.Get("/operations", h.Operations)
		r.Post("/operations", h.Launch)
		r.Get("/operations/{runID}", h.Operation)
		r.Post("/operations/{runID}/tracking", h.Track)
		r.Post("/operations/{runID}/tracking/revision", h.ReviseTracking)
		r.Post("/operations/{runID}/passage", h.GrantPassage)
		r.Post("/operations/{runID}/receive", h.Receive)
		r.Post("/operations/{runID}/complete", h.Complete)
		r.Delete("/operations/{runID}", h.Cancel)
		r.Get("/pigs", h.Pigs)
		r.Post("/pigs", h.RegisterPig)
		r.Get("/pigs/{pigID}", h.Pig)
		r.Put("/pigs/{pigID}", h.UpdatePig)
		r.Post("/pigs/{pigID}/components", h.InstallComponent)
		r.Put("/pigs/{pigID}/components/{componentID}", h.ReplaceComponent)
		r.Delete("/pigs/{pigID}/components/{componentID}", h.RemoveComponent)
		r.Post("/pigs/{pigID}/assessments", h.AssessPig)
		r.Get("/pigs/{pigID}/assessments", h.PigAssessments)
		r.Get("/equipment", h.Equipment)
		r.Get("/interlocks", h.Interlocks)
		r.Get("/incidents", h.Incidents)
		r.Get("/snapshot", h.Snapshot)
		r.Post("/snapshot", h.PersistSnapshot)
		r.Get("/recovery/barrier", h.RecoveryBarrier)
	})
	return router
}
