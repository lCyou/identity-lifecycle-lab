package api

import (
	"encoding/json"
	"errors"
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

	e := h.store.CreateEntity(req.Name)
	writeJSON(w, http.StatusCreated, e)
}

func (h *handler) listEntities(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.store.ListEntities())
}

func (h *handler) getEntity(w http.ResponseWriter, r *http.Request) {
	e, err := h.store.GetEntity(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
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

	e, err := h.store.Transition(id, identity.State(req.To), req.Actor, req.Reason)
	if err != nil {
		switch {
		case errors.Is(err, identity.ErrEntityNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, identity.ErrInvalidTransition):
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func (h *handler) listTransitions(w http.ResponseWriter, r *http.Request) {
	hist, err := h.store.History(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, hist)
}
