package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"gopkg.in/yaml.v3"
)

type reflectionSpec struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

type capabilitiesSpec struct {
	Endpoints []endpoint `json:"endpoints" yaml:"endpoints"`
}

type endpoint struct {
	Path        string   `json:"path" yaml:"path"`
	Methods     []string `json:"methods,omitempty" yaml:"methods"`
	ContentType string   `json:"contentType,omitempty" yaml:"contentType"`
	Description string   `json:"description,omitempty" yaml:"description"`
}

const capabilitiesDescription = `
  endpoints:
    - path: "/*"
      methods: [any]
      contentType: any
      description: |
        Requests to any URL that is not defined as an endpoint are accepted if they are valid.
        The response to any such request will be a status 200 without a body.
    - path: /capabilities
      methods: [GET]
      contentType: "-"
      description: | 
        Returns a JSON document describing the capabilities of albedo, i.e., available endpoints and their functions.
        If the query parameter 'quiet' is set to 'true', the response will only contain the path property for each
        endpoint.
    - path: /reflect
      methods: [POST]
      contentType: application/json
      description: |
        This endpoint responds according to the received specification.
        The specification is a JSON document with the following fields:

          status [integer]: the status code to respond with
          headers [map of header definitions]: the headers to respond with
          body [base64-encoded string]: body of the response, base64-encoded

        While this endpoint essentially allows for freeform responses, some restrictions apply:
          - responses with status code 1xx don't have a body; if you specify a body together with a 1xx
            status code, the behavior is undefined
          - response status codes must lie in the range of [100,599]
          - the names and values of headers must be valid according to the HTTP specification;
            invalid headers will be dropped
`

func Start(binding string, port int) *http.Server {
	server := &http.Server{
		Addr: fmt.Sprintf("%s:%d", binding, port),
	}

	http.HandleFunc("/", handleDefault)
	http.HandleFunc("/capabilities", handleCapabilities)
	http.HandleFunc("/capabilities/", handleCapabilities)
	http.HandleFunc("POST /reflect", handleReflect)
	http.HandleFunc("POST /reflect/", handleReflect)

	log.Fatal(server.ListenAndServe())
	return server
}

// Respond with empty 200 for all requests by default
func handleDefault(w http.ResponseWriter, r *http.Request) {}

func handleReflect(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Failed to parse request body"))
		return
	}
	spec := &reflectionSpec{}
	if err = json.Unmarshal(body, spec); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid JSON in request body"))
		return
	}

	for name, value := range spec.Headers {
		w.Header().Add(name, value)
	}
	if spec.Status < 100 || spec.Status >= 600 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Invalid status code: %d", spec.Status)))
		return
	}
	w.WriteHeader(spec.Status)

	decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(spec.Body))
	bodyBytes, err := io.ReadAll(decoder)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid base64 encoding of response body"))
		return

	}
	w.Write(bodyBytes)
}

func handleCapabilities(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")

	spec := &capabilitiesSpec{}
	err := yaml.Unmarshal([]byte(capabilitiesDescription), spec)
	if err != nil {
		log.Fatal("Failed to unmarshal capabilities description")
	}

	if r.URL.Query().Get("quiet") == "true" {
		quietSpec := &capabilitiesSpec{}
		for _, ep := range spec.Endpoints {
			quietSpec.Endpoints = append(quietSpec.Endpoints, endpoint{Path: ep.Path})
		}
		spec = quietSpec
	}

	body, err := json.Marshal(spec)
	if err != nil {
		log.Fatal("Failed to marshal capabilities")
	}
	w.Write(body)
}
