package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"stockticker/internal/controller"

	"github.com/gin-gonic/gin"
)

type Server struct {
	stockCtrler *controller.StockController
	httpServer  *http.Server
}

func NewServer(stockCtrler *controller.StockController, ip string, port int) (*Server, error) {
	return &Server{
		stockCtrler: stockCtrler,

		// https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/
		httpServer: &http.Server{
			Addr:         fmt.Sprintf("%s:%d", ip, port),
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
	}, nil
}

func (s *Server) Start(ctx context.Context) {
	s.httpServer.Handler = s.setupRouter(false)

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not start server: %v\n", err)
		}
	}()
}

func (s *Server) Stop(ctx context.Context, maxReqDuration time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, maxReqDuration)
	defer cancel()
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return err
	}
	return nil
}

func (s *Server) setupRouter(withLoggingAndMiddlware bool) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	var router *gin.Engine
	if withLoggingAndMiddlware {
		router = gin.Default()
	} else {
		router = gin.New()
	}

	router.LoadHTMLGlob("templates/*")

	router.GET("/", s.stock)

	api := router.Group("/api")
	v1 := api.Group("/v1")

	v1.GET("/liveness", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	v1.GET("/readiness", func(c *gin.Context) {
		// TODO: If Redis is necessary to avoid rate limits then this should probably fail if Redis is offline
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	return router
}

func (s *Server) stock(c *gin.Context) {
	viewData, err := s.stockCtrler.Stock(c.Request.Context())
	if err != nil {
		// TODO: This might be better as a metric if this service will see a high request volume
		log.Errorf("Failed to retrieve stock data: %v", err)
		// TODO: Who are the users of this service? Is it safe and/or useful (e.g. rate limiting) to expose more detail to them?
		c.HTML(http.StatusInternalServerError, "500.tmpl", gin.H{})
		return
	}
	c.HTML(http.StatusOK, s.stockCtrler.ViewTemplate(), viewData)
}
