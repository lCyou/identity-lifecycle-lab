package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/lCyou/identity-lifecycle-lab/internal/identity"
)

type handler struct {
	store *identity.Store
}

type createEntityRequest struct {
	Name string `json:"name"`
}

type transitionRequest struct {
	To     string `json:"to"`
	Actor  string `json:"actor"`
	Reason string `json:"reason"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeInternalError(w http.ResponseWriter, err error) {
	log.Printf("internal error: %v", err)
	writeError(w, http.StatusInternalServerError, "internal error")
}

func (h *handler) createEntity(w http.ResponseWriter, r *http.Request) {
	var req createEntityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	e, err := h.store.CreateEntity(r.Context(), req.Name)
	if err != nil {
		writeInternalError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

func (h *handler) listEntities(w http.ResponseWriter, r *http.Request) {
	list, err := h.store.ListEntities(r.Context())
	if err != nil {
		writeInternalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *handler) getEntity(w http.ResponseWriter, r *http.Request) {
	e, err := h.store.GetEntity(r.Context(), r.PathValue("id"))
	if err != nil {
		if errors.Is(err, identity.ErrEntityNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeInternalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func (h *handler) createTransition(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req transitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if req.To == "" || req.Actor == "" {
		writeError(w, http.StatusBadRequest, "to and actor are required")
		return
	}

	e, err := h.store.Transition(r.Context(), id, identity.State(req.To), req.Actor, req.Reason)
	if err != nil {
		switch {
		case errors.Is(err, identity.ErrEntityNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, identity.ErrInvalidTransition):
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeInternalError(w, err)
		}
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func (h *handler) listTransitions(w http.ResponseWriter, r *http.Request) {
	hist, err := h.store.History(r.Context(), r.PathValue("id"))
	if err != nil {
		if errors.Is(err, identity.ErrEntityNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeInternalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, hist)
}
