package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

const (
	readHeaderTimeout = 5 * time.Second
	shutdownTimeout   = 5 * time.Second
)

type Server struct {
	srv *http.Server
}

func New(addr string, handler http.Handler) *Server {
	return &Server{
		srv: &http.Server{
			Addr:              addr,
			Handler:           handler,
			ReadHeaderTimeout: readHeaderTimeout,
		},
	}
}

func (s *Server) Run(ctx context.Context) error {
	errs := make(chan error, 1)

	go func() {
		errs <- s.srv.ListenAndServe()
	}()

	select {
	case err := <-errs:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := s.srv.Shutdown(shutdownCtx); errors.Is(err, context.DeadlineExceeded) {
		slog.Warn("requests did not finish in time, closing connections", "timeout", shutdownTimeout)

		if err := s.srv.Close(); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	if err := <-errs; !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}
