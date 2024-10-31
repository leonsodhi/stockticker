package monitoring

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"

	log "github.com/sirupsen/logrus"
)

type PprofServer struct {
	httpServer *http.Server
}

func NewPprofServer(ip string, port int) (*PprofServer, error) {
	server := &PprofServer{
		httpServer: &http.Server{
			Addr: fmt.Sprintf("%s:%d", ip, port),
		},
	}
	return server, nil
}

func (ps *PprofServer) Listen() error {
	go func() {
		err := ps.httpServer.ListenAndServe()
		if err != nil {
			log.Fatalf("Could not start pprof server: %v\n", err)
		}
	}()
	return nil
}
