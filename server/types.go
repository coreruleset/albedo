package server

type CapabilitiesSpec struct {
	Endpoints []endpoint `json:"endpoints" yaml:"endpoints"`
}
