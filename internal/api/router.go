package api

import (
	"net/http"

	"github.com/lCyou/identity-lifecycle-lab/internal/identity"
)

// NewRouter はエンティティのライフサイクル操作用REST APIを構築する。
func NewRouter(store *identity.Store) http.Handler {
	h := &handler{store: store}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /entities", h.createEntity)
	mux.HandleFunc("GET /entities", h.listEntities)
	mux.HandleFunc("GET /entities/{id}", h.getEntity)
	mux.HandleFunc("POST /entities/{id}/transitions", h.createTransition)
	mux.HandleFunc("GET /entities/{id}/transitions", h.listTransitions)

	return mux
}
