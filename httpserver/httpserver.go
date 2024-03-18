//  Copyright (C) 2021-2023 Chronicle Labs, Inc.
//
//  This program is free software: you can redistribute it and/or modify
//  it under the terms of the GNU Affero General Public License as
//  published by the Free Software Foundation, either version 3 of the
//  License, or (at your option) any later version.
//
//  This program is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of
//  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//  GNU Affero General Public License for more details.
//
//  You should have received a copy of the GNU Affero General Public License
//  along with this program.  If not, see <http://www.gnu.org/licenses/>.

package httpserver

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/chronicleprotocol/go-utils/supervisor"
)

const shutdownTimeout = 1 * time.Second

type Middleware interface {
	Handle(http.Handler) http.Handler
}

type MiddlewareFunc func(http.Handler) http.Handler

func (m MiddlewareFunc) Handle(h http.Handler) http.Handler {
	return m(h)
}

type Service interface {
	supervisor.Service
	SetHandler(string, http.Handler)
	Addr() net.Addr
}

type NullServer struct {
	waitCh chan error
}

func (s *NullServer) Start(ctx context.Context) error {
	if s.waitCh != nil {
		return errors.New("service can be started only once")
	}
	if ctx == nil {
		return errors.New("context must not be nil")
	}
	s.waitCh = make(chan error)
	go func() {
		<-ctx.Done()
		close(s.waitCh)
	}()
	return nil
}

func (s *NullServer) SetHandler(string, http.Handler) {}
func (s *NullServer) Addr() net.Addr                  { return nil }
func (s *NullServer) Wait() <-chan error              { return s.waitCh }

// HTTPServer wraps the default net/http server to add the ability to use
// middlewares and support for the supervisor.Service interface.
type HTTPServer struct {
	ctx       context.Context
	ctxCancel context.CancelFunc
	serveCh   chan error
	waitCh    chan error

	handler map[string]http.Handler
	ln      net.Listener
	srv     *http.Server
}

// New creates a new HTTPServer instance.
func New(srv *http.Server) *HTTPServer {
	s := &HTTPServer{
		serveCh: make(chan error),
		waitCh:  make(chan error),
		handler: make(map[string]http.Handler),
		srv:     srv,
	}
	if srv.Handler != nil {
		s.handler[""] = srv.Handler
	}
	srv.Handler = http.HandlerFunc(s.serveHTTP)
	return s
}

// Use adds a middleware. Middlewares will be called in the reverse order
// they were added.
func (s *HTTPServer) Use(m ...Middleware) {
	for p := range s.handler {
		for _, m := range m {
			s.handler[p] = m.Handle(s.handler[p])
		}
	}
}

// SetHandler sets the handler for the server.
func (s *HTTPServer) SetHandler(path string, handler http.Handler) {
	s.handler[strings.Trim(path, "/")+"/"] = handler
}

// ServeHTTP prepares middlewares stack if necessary and calls ServerHTTP
// on the wrapped server.
func (s *HTTPServer) serveHTTP(rw http.ResponseWriter, r *http.Request) {
	for p, h := range s.handler {
		if strings.HasPrefix(strings.Trim(r.URL.Path, "/")+"/", p) {
			h.ServeHTTP(rw, r)
			return
		}
	}
	rw.WriteHeader(http.StatusNotFound)
}

// Start implements the supervisor.Service interface. It starts HTTP server.
func (s *HTTPServer) Start(ctx context.Context) error {
	if s.ctx != nil {
		return errors.New("service can be started only once")
	}
	if ctx == nil {
		return errors.New("context must not be nil")
	}
	s.ctx, s.ctxCancel = context.WithCancel(ctx)
	addr := s.srv.Addr
	if addr == "" {
		addr = ":http"
	}
	ln, err := (&net.ListenConfig{}).Listen(s.ctx, "tcp", addr)
	if err != nil {
		return err
	}
	s.ln = ln
	go s.shutdownHandler()
	go s.serve()
	return nil
}

// Wait implements the supervisor.Service interface.
func (s *HTTPServer) Wait() <-chan error {
	return s.waitCh
}

// Addr returns the server's network address.
func (s *HTTPServer) Addr() net.Addr {
	if s.ln == nil {
		return nil
	}
	return s.ln.Addr()
}

func (s *HTTPServer) serve() {
	select {
	case <-s.ctx.Done():
	case s.serveCh <- s.srv.Serve(s.ln):
	default:
	}
}

func (s *HTTPServer) shutdownHandler() {
	defer func() { close(s.waitCh) }()
	select {
	case <-s.ctx.Done():
		ctx, ctxCancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer ctxCancel()
		s.waitCh <- s.srv.Shutdown(ctx)
	case err := <-s.serveCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.waitCh <- err
		}
	}
}
