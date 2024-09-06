package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"io"
	"main/mesh"
	"net/http"
	"net/http/httptest"
	"testing"
)

var mockStartRequest = mesh.StartRequest{
	Command:              "echo Hello, World!",
	EnvironmentVariables: map[string]string{"TEST_VAR": "test_value"},
}

func TestStart(t *testing.T) {
	routes := mesh.NewRoutes()

	// create request body
	reqBody, err := json.Marshal(mockStartRequest)
	if err != nil {
		t.Fatal(err)
	}

	handle := handleWithChi(http.MethodPost, "/start", routes.Start(), bytes.NewBuffer(reqBody))

	var buff bytes.Buffer
	err = json.NewEncoder(&buff).Encode("Hello, World!\n")
	if err != nil {
		t.Fatal(err)
	}

	resp := handle("/start")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

}

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
		req.Header.Set(mesh.HdrRequestsContext, MockXRequestsContextString())
		req.Header.Set(mesh.HdrClientContext, MockID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		return w.Result()
	}
}
