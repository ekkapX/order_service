package api

import (
	"context"
	"database/sql"
	"net/http"

	"l0/internal/cache"
	"l0/internal/db"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Server struct {
	cache      *cache.Cache
	dbConn     *sql.DB
	logger     *zap.Logger
	router     *gin.Engine
	httpServer *http.Server
}

func NewServer(cache *cache.Cache, dbConn *sql.DB, logger *zap.Logger) *Server {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	if err := r.SetTrustedProxies(nil); err != nil {
		logger.Error("Failed to set trusted proxies", zap.Error(err))
	}
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	server := &Server{
		cache:  cache,
		dbConn: dbConn,
		logger: logger,
		router: r,
	}
	server.setupRoutes()
	return server
}

func (s *Server) setupRoutes() {
	s.router.Static("/web", "./web")
	s.router.GET("/", func(c *gin.Context) {
		c.File("./web/index.html")
	})
	s.router.GET("/order/:order_uid", s.handleGetOrder)
}

func (s *Server) handleGetOrder(c *gin.Context) {
	orderUID := c.Param("order_uid")
	if orderUID == "" {
		s.logger.Warn("Missing order_uid parameter")
		c.JSON(http.StatusBadRequest, gin.H{"error": "order_uid is required"})
		return
	}

	order, err := s.cache.GetOrder(c.Request.Context(), orderUID)
	if err == nil && order != nil {
		s.logger.Warn("Order retrieved from cache", zap.String("order_uid", orderUID))
		c.JSON(http.StatusOK, order)
		return
	}
	s.logger.Warn("Order not found in cache, cheking DB", zap.String("order_uid", orderUID), zap.Error(err))

	dbOrder, err := db.GetOrder(s.dbConn, orderUID, s.logger)
	if err != nil {
		s.logger.Warn("Order not found in DB", zap.String("order_uid", orderUID), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}

	order = &dbOrder

	if err := s.cache.SaveOrder(c.Request.Context(), *order); err != nil {
		s.logger.Error("Failed to cache order", zap.Error(err), zap.String("order_uid", orderUID))
	}

	s.logger.Info("Order retrieved from DB and cached", zap.String("order_uid", orderUID))
	c.JSON(http.StatusOK, order)
}

func (s *Server) Start(addr string) error {
	s.httpServer = &http.Server{Addr: addr, Handler: s.router}
	s.logger.Info("Starting HTTP server", zap.String("address", addr))

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down HTTP server...")
	return s.httpServer.Shutdown(ctx)
}
