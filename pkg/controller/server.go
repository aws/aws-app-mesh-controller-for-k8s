package controller

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Handler struct {
	http.ServeMux
}

func newHandler() *Handler {
	h := &Handler{}
	h.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "ok")
	})
	h.Handle("/metrics", promhttp.Handler())
	return h
}

func NewServer(cfg ServerOptions) *http.Server {
	return &http.Server{
		Handler:      newHandler(),
		Addr:         cfg.Address,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
}
