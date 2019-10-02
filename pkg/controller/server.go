package controller

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Handler struct {
	http.ServeMux
}

func newHandler() *http.ServeMux {
	mux := http.DefaultServeMux
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "ok")
	})
	mux.Handle("/metrics", promhttp.Handler())
	return mux
}

func NewServer(cfg ServerOptions) *http.Server {
	return &http.Server{
		Handler: newHandler(),
		Addr:    cfg.Address,
	}
}
