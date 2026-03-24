package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nicroldan/ans/registry/internal/handlers"
	"github.com/nicroldan/ans/registry/internal/search"
	"github.com/nicroldan/ans/registry/internal/store"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	esURL := os.Getenv("ELASTICSEARCH_URL")
	if esURL == "" {
		esURL = "http://localhost:9200"
	}

	adminToken := os.Getenv("REGISTRY_ADMIN_TOKEN")
	if adminToken == "" {
		b := make([]byte, 32)
		_, _ = rand.Read(b)
		adminToken = "adm_" + hex.EncodeToString(b)
		os.Setenv("REGISTRY_ADMIN_TOKEN", adminToken)
	}
	log.Printf("Admin token: %s", adminToken)

	memStore := store.New()

	mux := http.NewServeMux()

	// Try to connect to Elasticsearch for search capabilities.
	var engine *search.Engine
	eng, err := search.NewEngine([]string{esURL})
	if err != nil {
		log.Printf("WARNING: Failed to create Elasticsearch client: %v", err)
		log.Println("Search and sync endpoints will be DISABLED")
	} else {
		// Verify connectivity with a ping.
		pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := eng.Ping(pingCtx); err != nil {
			log.Printf("WARNING: Elasticsearch unavailable at %s: %v", esURL, err)
			log.Println("Search and sync endpoints will be DISABLED")
		} else {
			engine = eng
		}
		pingCancel()
	}

	if engine != nil {
		// Ensure the product index exists.
		idxCtx, idxCancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := engine.EnsureIndex(idxCtx); err != nil {
			log.Printf("WARNING: Failed to ensure Elasticsearch index: %v", err)
			log.Println("Search and sync endpoints will be DISABLED")
			engine = nil
		}
		idxCancel()
	}

	// Register store handler (pass engine as ProductDeleter, nil if ES unavailable).
	var deleter handlers.ProductDeleter
	if engine != nil {
		deleter = engine
	}
	storeHandler := handlers.NewStoreHandler(memStore, deleter)
	storeHandler.RegisterRoutes(mux)

	if engine != nil {
		syncHandler := handlers.NewSyncHandler(engine, memStore)
		syncHandler.RegisterRoutes(mux)

		searchHandler := handlers.NewSearchHandler(engine)
		searchHandler.RegisterRoutes(mux)

		log.Printf("Search ENABLED (Elasticsearch at %s)", esURL)
	} else {
		log.Println("Search DISABLED — registry running without Elasticsearch")
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	// Graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("Registry server listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Shutdown error: %v", err)
	}

	log.Println("Server stopped")
}
