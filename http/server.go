/*
 * Copyright (C) 2015 Red Hat, Inc.
 *
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *  http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 *
 */

package http

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/abbot/go-http-auth"
	gcontext "github.com/gorilla/context"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/skydive-project/skydive/common"
	"github.com/skydive-project/skydive/logging"
	"github.com/skydive-project/skydive/rbac"
)

// PathPrefix describes the prefix of the path of an URL
type PathPrefix string

// Route describes an HTTP route with a name, a HTTP verb,
// a path protected by an authentication backend
type Route struct {
	Name        string
	Method      string
	Path        interface{}
	HandlerFunc auth.AuthenticatedHandlerFunc
}

// Server describes a HTTP server for a service that dispatches requests to routes
type Server struct {
	sync.RWMutex
	http.Server
	Host        string
	ServiceType common.ServiceType
	Router      *mux.Router
	Addr        string
	Port        int
	lock        sync.Mutex
	listener    net.Listener
	wg          sync.WaitGroup
}

func copyRequestVars(old, new *http.Request) {
	kv := gcontext.GetAll(old)
	for k, v := range kv {
		gcontext.Set(new, k, v)
	}
}

// SetTLSHeader set TLS specific headers in the response
func SetTLSHeader(w http.ResponseWriter, r *http.Request) {
	if r.TLS != nil {
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
	}
}

// RegisterRoutes registers a set of routes protected by an authentication backend
func (s *Server) RegisterRoutes(routes []Route, auth AuthenticationBackend) {
	for _, route := range routes {
		r := s.Router.
			Methods(route.Method).
			Name(route.Name).
			Handler(auth.Wrap(route.HandlerFunc))
		switch p := route.Path.(type) {
		case string:
			r.Path(p)
		case PathPrefix:
			r.PathPrefix(string(p))
		}
	}
}

// Listen starts listening for TCP requests
func (s *Server) Listen() error {
	listenAddrPort := fmt.Sprintf("%s:%d", s.Addr, s.Port)
	ln, err := net.Listen("tcp", listenAddrPort)
	if err != nil {
		return fmt.Errorf("Failed to listen on %s:%d: %s", s.Addr, s.Port, err)
	}

	s.listener = ln
	logging.GetLogger().Infof("Listening on socket %s:%d", s.Addr, s.Port)
	return nil
}

// ListenAndServe starts listening and serving HTTP requests
func (s *Server) ListenAndServe() {
	if err := s.Listen(); err != nil {
		logging.GetLogger().Critical(err)
	}

	go s.Serve()
}

// Serve HTTP request
func (s *Server) Serve() {
	defer s.wg.Done()
	s.wg.Add(1)

	s.Handler = handlers.CompressHandler(s.Router)

	var err error
	if s.TLSConfig != nil {
		err = s.Server.ServeTLS(s.listener, "", "")
	} else {
		err = s.Server.Serve(s.listener)
	}

	if err == http.ErrServerClosed {
		return
	}
	logging.GetLogger().Errorf("Failed to serve on %s:%d: %s", s.Addr, s.Port, err)
}

// Unauthorized returns a 401 response
func Unauthorized(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte("401 Unauthorized\n"))
}

// Stop the server
func (s *Server) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.Server.Shutdown(ctx); err != nil {
		logging.GetLogger().Error("Shutdown error :", err)
	}
	s.listener.Close()
	s.wg.Wait()
}

// HandleFunc specifies the handler function and the authentication backend used for a given path
func (s *Server) HandleFunc(path string, f auth.AuthenticatedHandlerFunc, authBackend AuthenticationBackend) {
	postAuthHandler := func(w http.ResponseWriter, r *auth.AuthenticatedRequest) {
		// re-add user to its group
		if roles := rbac.GetUserRoles(r.Username); len(roles) == 0 {
			rbac.AddRoleForUser(r.Username, authBackend.DefaultUserRole(r.Username))
		}

		// re-send the permissions
		setPermissionsCookie(w, r.Username)

		f(w, r)
	}

	preAuthHandler := authBackend.Wrap(postAuthHandler)

	s.Router.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		// set tls headers first
		SetTLSHeader(w, r)

		preAuthHandler(w, r)
	})
}

// NewServer returns a new HTTP service for a service
func NewServer(host string, serviceType common.ServiceType, addr string, port int, tlsConfig *tls.Config) *Server {
	router := mux.NewRouter().StrictSlash(true)
	router.Headers("X-Host-ID", host, "X-Service-Type", serviceType.String())

	return &Server{
		Server: http.Server{
			TLSConfig: tlsConfig,
		},
		Host:        host,
		ServiceType: serviceType,
		Router:      router,
		Addr:        addr,
		Port:        port,
	}
}
