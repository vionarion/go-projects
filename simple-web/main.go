// simple-web is a tiny JSON API on the standard library only.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

type Person struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

// server keeps people in memory; the mutex makes handlers safe for concurrent requests.
type server struct {
	mu     sync.RWMutex
	people []Person
}

func newMux(s *server) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /people", s.list)
	mux.HandleFunc("POST /people", s.add)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	return mux
}

func (s *server) list(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	people := s.people
	if people == nil {
		people = []Person{}
	}
	writeJSON(w, http.StatusOK, people)
}

func (s *server) add(w http.ResponseWriter, r *http.Request) {
	var p Person
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&p); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if p.Name == "" || p.Age < 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required, age must be >= 0"})
		return
	}
	s.mu.Lock()
	s.people = append(s.people, p)
	s.mu.Unlock()
	writeJSON(w, http.StatusCreated, p)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func main() {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = "localhost:8080"
	}
	log.Printf("listening on %s", addr) // #nosec G706 -- addr comes from the operator's own environment
	srv := &http.Server{
		Addr:              addr,
		Handler:           newMux(&server{}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}
