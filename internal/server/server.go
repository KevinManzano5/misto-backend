// Package server handles internal server logic and routing
package server

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KevinManzano5/misto-backend/internal/middlewares"
)

// Server represents the main HTTP server configuration and state
type Server struct {
	httpServer *http.Server
	mux        *http.ServeMux
	logger     *slog.Logger
}

// New create and return a pointer of a server
func New(addr string, options ...Option) *Server {
	config := defaultConfig()

	for _, option := range options {
		option(&config)
	}

	mux := http.NewServeMux() // * HTTP Router

	httpServer := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadTimeout:       config.readTimeout,
		WriteTimeout:      config.writeTimeout,
		IdleTimeout:       config.idleTimeout,
		ReadHeaderTimeout: config.readHeaderTimeout,
	}

	return &Server{
		httpServer: httpServer,
		mux:        mux,
		logger:     config.logger,
	}
}

// Start initializes HTTP server
func (server *Server) Start() error {
	quit := make(chan os.Signal, 1)

	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	defer signal.Stop(quit)

	serveErr := make(chan error, 1)

	go func() {
		server.logger.Info("Server started", slog.String("addr", server.httpServer.Addr))

		if err := server.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serveErr <- err
		}
	}()

	select {
	case err := <-serveErr:
		return err
	case signal := <-quit:
		server.logger.Info("Signal received, shutting down server", slog.String("signal", signal.String()))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	defer cancel()

	if err := server.httpServer.Shutdown(ctx); err != nil {
		server.logger.Error("error in shutdown", slog.Any("error", err))

		return err
	}

	server.logger.Info("Server stopped successfully")

	return nil
}

// RegisterRoutes register routes
func (server *Server) RegisterRoutes(pattern string, handler http.Handler) {
	server.mux.Handle(pattern, handler)
}

// ServeHTTP server
func (server *Server) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	server.mux.ServeHTTP(writer, request)
}

// Use method
func (server *Server) Use(middleware func(http.Handler) http.Handler) {
	server.httpServer.Handler = middleware(server.httpServer.Handler)
}

// UseChain method
func (server *Server) UseChain(middlewaresParams ...middlewares.Middleware) {
	server.httpServer.Handler = middlewares.Chain(server.httpServer.Handler, middlewaresParams...)
}
