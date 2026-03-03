package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	addr := getenv("LISTEN_ADDR", ":8080")
	dbURL := os.Getenv("DB_URL")

	var store Store
	if dbURL != "" {
		s, err := NewPostgresStore(dbURL)
		if err != nil {
			log.Printf("postgres unavailable (%v), falling back to in-memory store", err)
			store = NewMemoryStore()
		} else {
			if err := s.Migrate("migrations"); err != nil {
				log.Printf("migration error: %v", err)
			}
			store = s
			log.Println("connected to postgres")
		}
	} else {
		log.Println("DB_URL not set, using in-memory store")
		store = NewMemoryStore()
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware)

	h := &Handlers{store: store}

	r.Route("/api", func(r chi.Router) {
		r.Route("/agent", func(r chi.Router) {
			r.Post("/enroll", h.Enroll)
			r.Post("/heartbeat", h.Heartbeat)
			r.Get("/policy", h.GetPolicy)
			r.Post("/events", h.IngestEvents)
			r.Get("/commands", h.GetCommands)
			r.Post("/commands/{id}/result", h.PostCommandResult)
		})
		r.Get("/devices", h.ListDevices)
		r.Get("/devices/{id}", h.GetDevice)
	})

	log.Printf("server listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
