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
	"time"

	"github.com/orcfax/oracle-suite/pkg/supervisor"
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
	SetHandler(http.Handler)
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

func (s *NullServer) Wait() <-chan error      { return s.waitCh }
func (s *NullServer) SetHandler(http.Handler) {}
func (s *NullServer) Addr() net.Addr          { return nil }

// HTTPServer wraps the default net/http server to add the ability to use
// middlewares and support for the supervisor.Service interface.
type HTTPServer struct {
	ctx       context.Context
	ctxCancel context.CancelFunc
	serveCh   chan error
	waitCh    chan error

	ln          net.Listener
	srv         *http.Server
	baseHandler http.Handler
	handler     http.Handler
}

// New creates a new HTTPServer instance.
func New(srv *http.Server) *HTTPServer {
	s := &HTTPServer{
		serveCh: make(chan error),
		waitCh:  make(chan error),
		srv:     srv,
	}
	s.baseHandler = srv.Handler
	s.handler = http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) { s.baseHandler.ServeHTTP(rw, r) })
	srv.Handler = http.HandlerFunc(s.ServeHTTP)
	return s
}

// Use adds a middleware. Middlewares will be called in the reverse order
// they were added.
func (s *HTTPServer) Use(m ...Middleware) {
	for _, m := range m {
		s.handler = m.Handle(s.handler)
	}
}

// SetHandler sets the handler for the server.
func (s *HTTPServer) SetHandler(handler http.Handler) {
	s.baseHandler = handler
}

// ServeHTTP prepares middlewares stack if necessary and calls ServerHTTP
// on the wrapped server.
func (s *HTTPServer) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(rw, r)
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
	s.serveCh <- s.srv.Serve(s.ln)
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
