package server

import (
	"context"
	"fmt"
	"io/fs"
	"net"
	"net/http"

	"github.com/zmorgan/umpire/web"
)

// Server wraps an HTTP server that serves the review UI and API.
type Server struct {
	httpServer *http.Server
	listener   net.Listener
}

// New creates a new Server on the given port. Port 0 picks an available port.
func New(port int) (*Server, error) {
	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", addr, err)
	}

	mux := http.NewServeMux()

	staticFS, err := fs.Sub(web.StaticFS, "static")
	if err != nil {
		listener.Close()
		return nil, fmt.Errorf("sub fs: %w", err)
	}
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	srv := &Server{
		httpServer: &http.Server{Handler: mux},
		listener:   listener,
	}
	return srv, nil
}

// Port returns the port the server is listening on.
func (s *Server) Port() int {
	return s.listener.Addr().(*net.TCPAddr).Port
}

// URL returns the full URL to the server.
func (s *Server) URL() string {
	return fmt.Sprintf("http://localhost:%d", s.Port())
}

// Mux returns the underlying ServeMux for registering additional routes.
func (s *Server) Mux() *http.ServeMux {
	return s.httpServer.Handler.(*http.ServeMux)
}

// Serve starts serving requests. Blocks until the server is shut down.
func (s *Server) Serve() error {
	err := s.httpServer.Serve(s.listener)
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
