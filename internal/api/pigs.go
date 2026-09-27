package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/wyw14/solo-036/internal/service"
)

type assessRequest struct {
	RouteID string `json:"route_id"`
}

func (h *Handler) Pigs(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.runtime.PigLedger())
}

func (h *Handler) RegisterPig(w http.ResponseWriter, r *http.Request) {
	var request service.PigSpecRequest
	if err := decodeBody(w, r, &request); err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	registered, err := h.runtime.RegisterPig(request)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusCreated, registered)
}

func (h *Handler) Pig(w http.ResponseWriter, r *http.Request) {
	entry, err := h.runtime.PigLedgerEntry(chi.URLParam(r, "pigID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

func (h *Handler) UpdatePig(w http.ResponseWriter, r *http.Request) {
	var request service.PigSpecRequest
	if err := decodeBody(w, r, &request); err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	updated, err := h.runtime.UpdatePigSpec(chi.URLParam(r, "pigID"), request)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) InstallComponent(w http.ResponseWriter, r *http.Request) {
	var request service.PigComponentRequest
	if err := decodeBody(w, r, &request); err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	installed, err := h.runtime.InstallPigComponent(chi.URLParam(r, "pigID"), request)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusCreated, installed)
}

func (h *Handler) ReplaceComponent(w http.ResponseWriter, r *http.Request) {
	var request service.PigComponentRequest
	if err := decodeBody(w, r, &request); err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	replacement, err := h.runtime.ReplacePigComponent(chi.URLParam(r, "pigID"), chi.URLParam(r, "componentID"), request)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusOK, replacement)
}

func (h *Handler) RemoveComponent(w http.ResponseWriter, r *http.Request) {
	if err := h.runtime.RemovePigComponent(chi.URLParam(r, "pigID"), chi.URLParam(r, "componentID")); err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) AssessPig(w http.ResponseWriter, r *http.Request) {
	var request assessRequest
	if err := decodeBody(w, r, &request); err != nil {
		writeError(w, r, http.StatusBadRequest, err)
		return
	}
	assessment, err := h.runtime.AssessPig(chi.URLParam(r, "pigID"), request.RouteID)
	if err != nil {
		writeError(w, r, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusCreated, assessment)
}

func (h *Handler) PigAssessments(w http.ResponseWriter, r *http.Request) {
	assessments, err := h.runtime.PigAssessments(chi.URLParam(r, "pigID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, assessments)
}
