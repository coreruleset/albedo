package server

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"hash/maphash"
	"io"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

var capabilities *CapabilitiesSpec
var dynamicEndpointMutex = sync.RWMutex{}
var dynamicEndpoints = map[uint64]reflectionSpec{}
var dynamicEndpointsHash = maphash.Hash{}

//go:embed capabilities.yaml
var capabilitiesDescription []byte

func Start(binding string, port int) {
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", binding, port),
		Handler: Handler(),
	}

	slog.Debug("Starting server")
	err := server.ListenAndServe()
	slog.Info("Server stopped", "exit-status", err)
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
		slog.Info(fmt.Sprintf("Received default request to %s", r.URL))
	}
}

func handleReflect(w http.ResponseWriter, r *http.Request) {
	slog.Info("Received reflection request")

	slog.Debug("Reading body")
	body, err := io.ReadAll(r.Body)
	if slog.Default().Enabled(context.TODO(), slog.LevelDebug) {
		numBytes, unit := toHumanReadableMemorySize(uint64(len(body)))
		slog.Debug(fmt.Sprintf("Body size: %d%s", numBytes, unit))
	}

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, err = w.Write([]byte("Failed to parse request body"))
		if err != nil {
			slog.Warn("Failed to write response body", "error", err.Error())
		}
		slog.Warn("Failed to parse request body")
		return
	}
	slog.Debug("Parsing reflection specification")
	spec := &reflectionSpec{}
	if err = json.Unmarshal(body, spec); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, err = w.Write([]byte("Invalid JSON in request body"))
		if err != nil {
			slog.Warn("Failed to write response body", "error", err.Error())
		}
		slog.Warn("Invalid JSON in request body")
		return
	}

	doReflect(w, r, spec)

}

func handleCapabilities(w http.ResponseWriter, r *http.Request) {
	slog.Info("Received capabilities request")
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
		slog.Warn("Failed to write response body", "error", err.Error())
	}
}

func handleConfigureReflection(w http.ResponseWriter, r *http.Request) {
	slog.Info("Received configuration request")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, err = w.Write([]byte("Failed to parse request body"))
		if err != nil {
			slog.Warn("Failed to write response body", "error", err.Error())
		}
		slog.Info("Failed to parse request body")
		return
	}
	spec := &configureReflectionSpec{}
	if err = json.Unmarshal(body, spec); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, err = w.Write([]byte("Invalid JSON in request body"))
		if err != nil {
			slog.Warn("Failed to write response body", "error", err.Error())
		}
		slog.Info("Invalid JSON in request body")
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
	slog.Info("Received reset request. Discarding all endpoint configurations now")
	for k := range dynamicEndpoints {
		delete(dynamicEndpoints, k)
	}
}

func decodeBody(spec *reflectionSpec) (string, error) {
	slog.Debug("Decoding body")

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
	slog.Info(fmt.Sprintf("Reflecting response for '%s' request to '%s'", r.Method, r.RequestURI))

	if spec.LogMessage != "" {
		slog.Info(spec.LogMessage)
	}

	for name, value := range spec.Headers {
		slog.Info(fmt.Sprintf("Reflecting header '%s':'%s'", name, value))
		w.Header().Add(name, value)
	}

	if spec.Status > 0 && spec.Status < 100 || spec.Status >= 600 {
		w.WriteHeader(http.StatusBadRequest)
		_, err := fmt.Fprintf(w, "Invalid status code: %d", spec.Status)
		if err != nil {
			slog.Warn("Failed to write response body", "error", err.Error())
		}
		slog.Info(fmt.Sprintf("Invalid status code: %d", spec.Status))
		return
	}
	status := spec.Status
	if status == 0 {
		status = http.StatusOK
	}
	slog.Info(fmt.Sprintf("Reflecting status '%d'", status))
	w.WriteHeader(status)

	responseBody, err := decodeBody(spec)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, err = w.Write([]byte(err.Error()))
		if err != nil {
			slog.Warn("Failed to write response body", "error", err.Error())
		}
		slog.Info(err.Error())
		return
	}

	if responseBody == "" {
		return
	}

	responseBodyBytes := []byte(responseBody)
	if len(responseBody) > 200 {
		responseBody = responseBody[:min(len(responseBody), 200)] + "..."
	}
	slog.Info(fmt.Sprintf("Reflecting body '%s'", responseBody))
	_, err = w.Write(responseBodyBytes)
	if err != nil {
		slog.Warn("Failed to write response body", "error", err.Error())
	}
}

func getCapabilities() *CapabilitiesSpec {
	if capabilities != nil {
		return capabilities
	}

	spec := &CapabilitiesSpec{}
	err := yaml.Unmarshal(capabilitiesDescription, spec)
	if err != nil {
		slog.Error("Failed to unmarshal capabilities description")
	}

	capabilities = spec

	return capabilities
}

func toHumanReadableMemorySize(numBytes uint64) (uint64, string) {
	units := []string{"B", "KB", "MB", "GB"}
	unit := 0
	for numBytes > 10000 {
		numBytes /= 1000
		unit++
	}
	var selectedUnit string
	if unit < len(units) {
		selectedUnit = units[unit]
	} else {
		selectedUnit = "(too big...)"
	}
	return numBytes, selectedUnit
}
