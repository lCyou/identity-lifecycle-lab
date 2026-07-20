package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/lCyou/identity-lifecycle-lab/internal/api"
	"github.com/lCyou/identity-lifecycle-lab/internal/identity"
)

const (
	defaultDatabaseURL = "postgres://identity:identity@localhost:5432/identity_lifecycle?sslmode=disable"
	shutdownTimeout    = 10 * time.Second
	dbConnectTimeout   = 5 * time.Second
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = defaultDatabaseURL
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	pingCtx, cancel := context.WithTimeout(context.Background(), dbConnectTimeout)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		return err
	}
	log.Print("connected to database")

	store := identity.NewStore(db)
	router := api.NewRouter(store)

	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	serveErr := make(chan error, 1)
	go func() {
		log.Printf("identity-lifecycle-lab listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
			return
		}
		serveErr <- nil
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
		log.Print("shutdown signal received, draining in-flight requests")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return err
	}
	<-serveErr // ListenAndServeのgoroutineの終了を待つ

	log.Print("server stopped, closing database connection")
	return nil
}
