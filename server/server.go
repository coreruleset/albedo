package server

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"hash/maphash"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

var capabilities *CapabilitiesSpec
var dynamicEndpointMutex = sync.RWMutex{}
var dynamicEndpoints = map[uint64]reflectionSpec{}
var dynamicEndpointsHash = maphash.Hash{}

func Start(binding string, port int) *http.Server {
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", binding, port),
		Handler: Handler(),
	}

	log.Fatal(server.ListenAndServe())
	return server
}

func Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/", handleDefault)
	mux.HandleFunc("/capabilities", handleCapabilities)
	mux.HandleFunc("/capabilities/", handleCapabilities)
	mux.HandleFunc("POST /reflect", handleReflect)
	mux.HandleFunc("POST /reflect/", handleReflect)
	mux.HandleFunc("POST /configure_reflection", handleConfigureReflection)
	mux.HandleFunc("POST /configure_reflection/", handleConfigureReflection)
	mux.HandleFunc("PUT /reset", handleReset)
	mux.HandleFunc("PUT /reset/", handleReset)

	return mux
}

// Respond with empty 200 for all requests by default.
// If the request matches a configured dynamic endpoint, reflect as specified
// for that endpoint.
func handleDefault(w http.ResponseWriter, r *http.Request) {
	key := computeEndpointKey(r.Method, r.RequestURI)
	dynamicEndpointMutex.RLock()
	reflectionSpec, ok := dynamicEndpoints[key]
	dynamicEndpointMutex.RUnlock()

	if ok {
		doReflect(w, r, &reflectionSpec)
	} else {
		log.Printf("Received default request to %s", r.URL)
	}
}

func handleReflect(w http.ResponseWriter, r *http.Request) {
	log.Println("Received reflection request")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, err = w.Write([]byte("Failed to parse request body"))
		if err != nil {
			log.Printf("Failed to write response body: %s", err.Error())
		}
		log.Println("Failed to parse request body")
		return
	}
	spec := &reflectionSpec{}
	if err = json.Unmarshal(body, spec); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, err = w.Write([]byte("Invalid JSON in request body"))
		if err != nil {
			log.Printf("Failed to write response body: %s", err.Error())
		}
		log.Println("Invalid JSON in request body")
		return
	}

	doReflect(w, r, spec)

}

func handleCapabilities(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")

	spec := getCapabilities()
	if r.URL.Query().Get("quiet") == "true" {
		quietSpec := &CapabilitiesSpec{}
		for _, ep := range spec.Endpoints {
			quietSpec.Endpoints = append(quietSpec.Endpoints, endpoint{Path: ep.Path})
		}
		spec = quietSpec
	}

	body, err := json.Marshal(spec)
	if err != nil {
		log.Fatal("Failed to marshal capabilities")
	}

	_, err = w.Write(body)
	if err != nil {
		log.Printf("Failed to write response body: %s", err.Error())
	}
}

func handleConfigureReflection(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, err = w.Write([]byte("Failed to parse request body"))
		if err != nil {
			log.Printf("Failed to write response body: %s", err.Error())
		}
		log.Println("Failed to parse request body")
		return
	}
	spec := &configureReflectionSpec{}
	if err = json.Unmarshal(body, spec); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, err = w.Write([]byte("Invalid JSON in request body"))
		if err != nil {
			log.Printf("Failed to write response body: %s", err.Error())
		}
		log.Println("Invalid JSON in request body")
		return
	}

	dynamicEndpointMutex.Lock()
	defer dynamicEndpointMutex.Unlock()
	for _, _endpoint := range spec.Endpoints {
		key := computeEndpointKey(_endpoint.Method, _endpoint.Url)
		dynamicEndpoints[key] = spec.reflectionSpec
	}
}

func handleReset(w http.ResponseWriter, r *http.Request) {
	log.Println("Received reset request. Discarding all endpoint configurations now")
	for k := range dynamicEndpoints {
		delete(dynamicEndpoints, k)
	}
}

func decodeBody(spec *reflectionSpec) (string, error) {
	if spec.Body != "" {
		return spec.Body, nil
	}

	if spec.EncodedBody == "" {
		return "", nil
	}

	decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(spec.EncodedBody))
	bodyBytes, err := io.ReadAll(decoder)
	if err != nil {
		return "", errors.New("invalid base64 encoding of response body")

	}
	return string(bodyBytes), nil
}

func computeEndpointKey(method string, url string) uint64 {
	dynamicEndpointsHash.Reset()
	dynamicEndpointsHash.WriteString(method)
	dynamicEndpointsHash.WriteString(url)
	return dynamicEndpointsHash.Sum64()
}

func doReflect(w http.ResponseWriter, r *http.Request, spec *reflectionSpec) {
	log.Printf("Reflecting response for '%s' request to '%s'", r.Method, r.RequestURI)

	if spec.LogMessage != "" {
		log.Println(spec.LogMessage)
	}

	for name, value := range spec.Headers {
		log.Printf("Reflecting header '%s':'%s'", name, value)
		w.Header().Add(name, value)
	}

	if spec.Status > 0 && spec.Status < 100 || spec.Status >= 600 {
		w.WriteHeader(http.StatusBadRequest)
		_, err := fmt.Fprintf(w, "Invalid status code: %d", spec.Status)
		if err != nil {
			log.Printf("Failed to write response body: %s", err.Error())
		}
		log.Printf("Invalid status code: %d", spec.Status)
		return
	}
	status := spec.Status
	if status == 0 {
		status = http.StatusOK
	}
	log.Printf("Reflecting status '%d'", status)
	w.WriteHeader(status)

	responseBody, err := decodeBody(spec)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, err = w.Write([]byte(err.Error()))
		if err != nil {
			log.Printf("Failed to write response body: %s", err.Error())
		}
		log.Println(err.Error())
		return
	}

	if responseBody == "" {
		return
	}

	responseBodyBytes := []byte(responseBody)
	if len(responseBody) > 200 {
		responseBody = responseBody[:min(len(responseBody), 200)] + "..."
	}
	log.Printf("Reflecting body '%s'", responseBody)
	_, err = w.Write(responseBodyBytes)
	if err != nil {
		log.Printf("Failed to write response body: %s", err.Error())
	}
}

func getCapabilities() *CapabilitiesSpec {
	if capabilities != nil {
		return capabilities
	}

	spec := &CapabilitiesSpec{}
	file, err := os.Open("capabilities.yaml")
	if err != nil {
		log.Fatal("Failed to open capabilities description")
	}
	description, err := io.ReadAll(file)
	if err != nil {
		log.Fatal("Failed to read capabilities description")
	}
	err = yaml.Unmarshal(description, spec)
	if err != nil {
		log.Fatal("Failed to unmarshal capabilities description")
	}

	capabilities = spec

	return capabilities
}
