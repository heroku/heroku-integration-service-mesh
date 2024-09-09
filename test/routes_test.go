package test

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"io"
	"main/mesh"
	"net/http"
	"net/http/httptest"
)

type chiHandler func(path string) *http.Response

func handleWithChi(method, path string, h http.HandlerFunc, body io.Reader) chiHandler {
	r := chi.NewRouter()
	r.Use(middleware.GetHead)

	switch method {
	case http.MethodGet:
		r.Get(path, h)
	case http.MethodPost:
		r.Post(path, h)
	}
	return func(path string) *http.Response {
		req := httptest.NewRequest(method, fmt.Sprintf("http://localhost%s", path), body)
		req.Header.Set(mesh.HdrNameRequestID, MockOrgID)
		req.Header.Set(mesh.HdrRequestsContext, convertContextToString(MockValidXRequestsContext))
		req.Header.Set(mesh.HdrClientContext, MockID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		return w.Result()
	}
}
