package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"main/mesh"
	"net/http"
)

// Using chi router to build the HTTP routes
type RouteBuilder struct {
	router chi.Router
}

func NewRouteBuilder() *RouteBuilder {
	return &RouteBuilder{
		router: defaultRouter(),
	}
}

func (rb *RouteBuilder) Scope(prefix string, callback func(chi.Router)) {
	builder := &RouteBuilder{}
	rb.router.Route(prefix, func(r chi.Router) {
		builder.router = r
	})

	callback(builder.router)
}

func defaultRouter(middlewares ...func(http.Handler) http.Handler) *chi.Mux {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	for _, middleware := range middlewares {
		router.Use(middleware)
	}

	return router
}

func NewRouter() chi.Router {
	rb := NewRouteBuilder()

	rb.Scope("/", func(r chi.Router) {
		mesh.InitializeRoutes(r)
	})

	return rb.router
}
