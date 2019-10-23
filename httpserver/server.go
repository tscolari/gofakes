package httpserver

import (
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/pkg/errors"
)

type Server struct {
	listener    net.Listener
	responses   map[string]map[string]http.HandlerFunc
	requests    []*http.Request
	handlerStub http.HandlerFunc
	lock        sync.RWMutex
}

func New() *Server {
	return &Server{
		responses: map[string]map[string]http.HandlerFunc{},
		requests:  []*http.Request{},
		lock:      sync.RWMutex{},
	}
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return errors.Wrap(err, "creating listener")
	}

	s.listener = listener
	go http.Serve(s.listener, http.HandlerFunc(s.handleFunc))
	return nil
}

func (s *Server) Stop() error {
	return s.listener.Close()
}

func (s *Server) Reset() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.responses = map[string]map[string]http.HandlerFunc{}
	s.requests = []*http.Request{}
	s.handlerStub = nil
}

func (s *Server) Addr() string {
	return "http://" + s.listener.Addr().String()
}

func (s *Server) RequestNum(index int) *http.Request {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.requests[index]
}

func (s *Server) RequestCount() int {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return len(s.requests)
}

func (s *Server) HandlerStub(handler http.HandlerFunc) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.handlerStub = handler
}

func (s *Server) RegisterPayload(method, path string, statusCode int, payload []byte) {
	handler := func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(statusCode)
		rw.Write(payload)
	}

	s.RegisterHandler(method, path, handler)
}

func (s *Server) RegisterHandler(method, path string, handler http.HandlerFunc) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, ok := s.responses[path]; !ok {
		s.responses[path] = map[string]http.HandlerFunc{}
	}

	s.responses[path][strings.ToLower(method)] = handler
}

func (s *Server) handleFunc(rw http.ResponseWriter, r *http.Request) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	s.requests = append(s.requests, r)

	if s.handlerStub != nil {
		s.handlerStub(rw, r)
		return
	}

	methods, ok := s.responses[r.URL.Path]
	if !ok {
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	handleFunc, ok := methods[strings.ToLower(r.Method)]
	if !ok {
		rw.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	handleFunc(rw, r)
}
