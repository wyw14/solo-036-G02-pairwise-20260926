package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/wyw14/solo-036/internal/model"
)

type pigRequest struct {
	ID           string               `json:"id"`
	Name         string               `json:"name"`
	BodyDiameter float64              `json:"body_diameter"`
	Length       float64              `json:"length"`
	MaxPressure  float64              `json:"max_pressure"`
	Components   []model.PigComponent `json:"components"`
}

type componentRequest struct {
	Kind     string  `json:"kind"`
	Diameter float64 `json:"diameter"`
	Material string  `json:"material"`
}

type compatibilityRequest struct {
	RouteID        string  `json:"route_id"`
	TargetPressure float64 `json:"target_pressure"`
}

func (h *Handler) Pigs(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.runtime.ListPigs())
}

func (h *Handler) RegisterPig(w http.ResponseWriter, r *http.Request) {
	var request pigRequest
	if err := decodeBody(w, r, &request); err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	spec, err := h.runtime.RegisterPig(model.PigSpec{
		ID:           request.ID,
		Name:         request.Name,
		BodyDiameter: request.BodyDiameter,
		Length:       request.Length,
		MaxPressure:  request.MaxPressure,
		Components:   request.Components,
	})
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusCreated, spec)
}

func (h *Handler) Pig(w http.ResponseWriter, r *http.Request) {
	spec, err := h.runtime.Pig(chi.URLParam(r, "pigID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, spec)
}

func (h *Handler) ReplacePigComponent(w http.ResponseWriter, r *http.Request) {
	var request componentRequest
	if err := decodeBody(w, r, &request); err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	spec, err := h.runtime.ReplacePigComponent(chi.URLParam(r, "pigID"), chi.URLParam(r, "componentID"), model.PigComponent{
		Kind:     model.ComponentKind(request.Kind),
		Diameter: request.Diameter,
		Material: request.Material,
	})
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, spec)
}

func (h *Handler) EvaluatePigCompatibility(w http.ResponseWriter, r *http.Request) {
	var request compatibilityRequest
	if err := decodeBody(w, r, &request); err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	conclusion, err := h.runtime.EvaluatePigCompatibility(chi.URLParam(r, "pigID"), request.RouteID, request.TargetPressure)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, conclusion)
}

func (h *Handler) PigCompatibility(w http.ResponseWriter, r *http.Request) {
	conclusions, err := h.runtime.PigCompatibility(chi.URLParam(r, "pigID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, conclusions)
}
