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
	Status     int               `json:"status"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	LogMessage string            `json:"logMessage"`
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

          status      [integer]: the status code to respond with
          headers     [map of header definitions]: the headers to respond with
          body        [base64-encoded string]: body of the response, base64-encoded
          logMessage  [string]: message to log for the request; useful for matching requests to tests

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
func handleDefault(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received default request to %s", r.URL)
}

func handleReflect(w http.ResponseWriter, r *http.Request) {
	log.Println("Received reflection request")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Failed to parse request body"))
		log.Println("Failed to parse request body")
		return
	}
	spec := &reflectionSpec{}
	if err = json.Unmarshal(body, spec); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid JSON in request body"))
		log.Println("Invalid JSON in request body")
		return
	}

	if spec.LogMessage != "" {
		log.Println(spec.LogMessage)
	}

	for name, value := range spec.Headers {
		log.Printf("Reflecting header '%s':'%s'", name, value)
		w.Header().Add(name, value)
	}

	if spec.Status > 0 && spec.Status < 100 || spec.Status >= 600 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Invalid status code: %d", spec.Status)))
		log.Printf("Invalid status code: %d", spec.Status)
		return
	}
	status := spec.Status
	if status == 0 {
		status = http.StatusOK
	}
	log.Printf("Reflecting status '%d'", status)
	w.WriteHeader(status)

	decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(spec.Body))
	bodyBytes, err := io.ReadAll(decoder)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid base64 encoding of response body"))
		log.Println("Invalid base64 encoding of response body")
		return

	}

	bodyString := string(bodyBytes)
	if len(bodyString) > 200 {
		bodyString = bodyString[:min(len(bodyString), 200)] + "..."
	}
	log.Printf("Reflecting body '%s'", bodyString)
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
