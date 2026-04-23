package app

import (
	"log"
	"net/http"
	"time"

	"zookeeper/internal/controller"
	"zookeeper/internal/router"
	"zookeeper/internal/service"
)

type Server struct {
	httpServer *http.Server
}

func NewServer(address string) *Server {
	healthService := service.NewHealthService()
	nodeService := service.NewNodeService()

	healthController := controller.NewHealthController(healthService)
	nodeController := controller.NewNodeController(nodeService)
	httpRouter := router.NewHTTPRouter(healthController, nodeController)

	return &Server{
		httpServer: &http.Server{
			Addr:              address,
			Handler:           httpRouter.Handler(),
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

func (s *Server) Start() error {
	log.Printf("zookeeper listening on %s", s.httpServer.Addr)
	err := s.httpServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
