package main

import (
	"log"
	"net/http"

	"github.com/lCyou/identity-lifecycle-lab/internal/api"
	"github.com/lCyou/identity-lifecycle-lab/internal/identity"
)

func main() {
	store := identity.NewStore()
	router := api.NewRouter(store)

	const addr = ":8080"
	log.Printf("identity-lifecycle-lab listening on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal(err)
	}
}
