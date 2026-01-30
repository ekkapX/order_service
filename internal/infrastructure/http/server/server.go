package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"l0/internal/infrastructure/http/handlers"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Server struct {
	Router     *gin.Engine
	httpServer *http.Server
	logger     *zap.Logger
}

func NewServer(orderHandler *handlers.OrderHandler, logger *zap.Logger) *Server {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	if err := r.SetTrustedProxies(nil); err != nil {
		logger.Error("Failed to set trusted proxies", zap.Error(err))
	}
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	server := &Server{
		logger: logger,
		Router: r,
	}
	server.setupRoutes(*orderHandler)
	return server
}

func (s *Server) setupRoutes(orderHandler handlers.OrderHandler) {
	s.Router.Static("/web", "./web")
	s.Router.GET("/", func(c *gin.Context) {
		c.File("./web/index.html")
	})

	s.Router.GET("/order/:order_uid", orderHandler.GetByUID)
}

func (s *Server) Start(addr string) error {
	s.httpServer = &http.Server{Addr: addr, Handler: s.Router, ReadHeaderTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second}
	s.logger.Info("Starting HTTP server", zap.String("address", addr))

	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down HTTP server...")
	return s.httpServer.Shutdown(ctx)
}
