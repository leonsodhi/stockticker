package monitoring

import (
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type PromServer struct {
	httpServer *http.Server
}

func NewPrometheusServer(ip string, port int) (*PromServer, error) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	server := &PromServer{
		httpServer: &http.Server{
			Addr:        fmt.Sprintf("%s:%d", ip, port),
			Handler:     mux,
			ReadTimeout: 30 * time.Second,
		},
	}
	return server, nil
}

func (ps *PromServer) Listen() error {
	go func() {
		err := ps.httpServer.ListenAndServe()
		if err != nil {
			log.Fatalf("Could not start prometheus metrics server: %v\n", err)
		}
	}()
	return nil
}
