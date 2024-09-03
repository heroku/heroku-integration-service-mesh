package mesh

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"io"
	"log/slog"
	"net/http"
)

type Routes struct {
	transport http.RoundTripper
}

type PassThroughResponse struct {
	Header *RequestHeader    `json:"header"`
	Body   map[string]string `json:"body"`
}

func InitializeRoutes(router chi.Router) {
	routes := NewRoutes()
	router.Post("/", routes.PassThrough())
}

func NewRoutes() *Routes {
	return &Routes{http.DefaultTransport}
}

func (route *Routes) PassThrough() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Validate the request
		requestHeader, err := ValidateRequest(r.Header)
		if err != nil {
			slog.Error("Invalid request %v", err)
			http.Error(w, "Invalid request: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Read request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			slog.Error("Error reading body %v", err)
			http.Error(w, "Error reading body: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// transform body into a json format
		var data map[string]string
		err = json.Unmarshal(body, &data)
		if err != nil {
			http.Error(w, "Error decoding JSON", http.StatusBadRequest)
			return
		}

		defer r.Body.Close()

		response := &PassThroughResponse{
			Header: requestHeader,
			Body:   data,
		}

		//convert entire response to JSON
		resp, err := json.Marshal(response)
		if err != nil {
			http.Error(w, "Error creating JSON response", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(resp)

	}
}
