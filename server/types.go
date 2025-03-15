package server

type CapabilitiesSpec struct {
	Endpoints []endpoint `json:"endpoints" yaml:"endpoints"`
}

type reflectionSpec struct {
	Status      int               `json:"status"`
	Headers     map[string]string `json:"headers"`
	Body        string            `json:"body"`
	EncodedBody string            `json:"encodedBody"`
	LogMessage  string            `json:"logMessage"`
}

type configureReflectionSpec struct {
	reflectionSpec
	Endpoints []dynamicEndpointSpec `json:"endpoints"`
}

type dynamicEndpointSpec struct {
	Method string `json:"method"`
	Url    string `json:"url"`
}

type endpoint struct {
	Path        string   `json:"path" yaml:"path"`
	Methods     []string `json:"methods,omitempty" yaml:"methods"`
	ContentType string   `json:"contentType,omitempty" yaml:"contentType"`
	Description string   `json:"description,omitempty" yaml:"description"`
}
