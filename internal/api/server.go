package api

import (
	"encoding/json"
	"net/http"

	"github.com/mezotov/netdiscover/internal/store"
)

type Server struct {
	store store.Store
}

func New(s store.Store) *Server {
	return &Server{s}
}

func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/devices", s.devices)

	return http.ListenAndServe(addr, mux)
}

func (s *Server) devices(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.store.All())
}
