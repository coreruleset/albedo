package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"
)

type serverTestSuite struct {
	suite.Suite
}

func TestServerTestSuite(t *testing.T) {
	suite.Run(t, new(serverTestSuite))
}

func (s *serverTestSuite) TestDefaultRequest() {
	server := httptest.NewServer((http.HandlerFunc)(handleDefault))
	s.T().Cleanup(server.Close)

	client := http.Client{}
	response, err := client.Get(server.URL)
	s.Require().NoError(err)

	s.Equal(response.StatusCode, http.StatusOK)
	s.Len(response.Header, 2)
	s.Contains(response.Header, "Date")
	s.Contains(response.Header, "Content-Length")
	s.Len(response.Header["Content-Length"], 1)
	s.Equal("0", response.Header["Content-Length"][0])
}

func (s *serverTestSuite) TestReflect_Body() {
	server := httptest.NewServer((http.HandlerFunc)(handleReflect))
	s.T().Cleanup(server.Close)

	responseBody := "a dummy body \t \n\r\r\n\r\n"
	spec := &reflectionSpec{
		Status: 202,
		Headers: map[string]string{
			"header1":  "value 1",
			"header_2": "value :2",
		},
		Body: responseBody,
	}
	body, err := json.Marshal(spec)
	s.Require().NoError(err)
	request, err := http.NewRequest("POST", server.URL+"/reflect", bytes.NewReader(body))
	s.Require().NoError(err)
	client := http.Client{}
	response, err := client.Do(request)
	s.Require().NoError(err)

	s.Equal(spec.Status, response.StatusCode)
	s.Len(response.Header, 5)
	s.Contains(response.Header, "Header1")
	s.Equal("value 1", response.Header["Header1"][0])
	s.Contains(response.Header, "Header_2")
	s.Equal("value :2", response.Header["Header_2"][0])

	reflectedBody, err := io.ReadAll(response.Body)
	s.Require().NoError(err)
	s.Equal(responseBody, string(reflectedBody))
}

func (s *serverTestSuite) TestReflect_EncodedBody() {
	server := httptest.NewServer((http.HandlerFunc)(handleReflect))
	s.T().Cleanup(server.Close)

	responseBodyString := "a dummy body \t \n\r\r\n\r\n"
	responseBody := base64.StdEncoding.EncodeToString([]byte(responseBodyString))
	spec := &reflectionSpec{
		Status: 202,
		Headers: map[string]string{
			"header1":  "value 1",
			"header_2": "value :2",
		},
		EncodedBody: responseBody,
	}
	body, err := json.Marshal(spec)
	s.Require().NoError(err)
	request, err := http.NewRequest("POST", server.URL+"/reflect", bytes.NewReader(body))
	s.Require().NoError(err)
	client := http.Client{}
	response, err := client.Do(request)
	s.Require().NoError(err)

	s.Equal(spec.Status, response.StatusCode)
	s.Len(response.Header, 5)
	s.Contains(response.Header, "Header1")
	s.Equal("value 1", response.Header["Header1"][0])
	s.Contains(response.Header, "Header_2")
	s.Equal("value :2", response.Header["Header_2"][0])

	reflectedBody, err := io.ReadAll(response.Body)
	s.Require().NoError(err)
	s.Equal(responseBodyString, string(reflectedBody))
}

func (s *serverTestSuite) TestReflect_LogMessage() {
	logBuffer := &bytes.Buffer{}
	log.SetOutput(logBuffer)

	server := httptest.NewServer((http.HandlerFunc)(handleReflect))
	s.T().Cleanup(server.Close)

	spec := &reflectionSpec{
		LogMessage: "a log message",
	}
	body, err := json.Marshal(spec)
	s.Require().NoError(err)
	request, err := http.NewRequest("POST", server.URL+"/reflect", bytes.NewReader(body))
	s.Require().NoError(err)
	client := http.Client{}
	_, err = client.Do(request)
	s.Require().NoError(err)

	output, err := io.ReadAll(logBuffer)
	s.Require().NoError(err)

	s.Require().NoError(err)
	s.Contains(string(output), spec.LogMessage)
}

func (s *serverTestSuite) TestCapabilities() {
	server := httptest.NewServer((http.HandlerFunc)(handleCapabilities))
	s.T().Cleanup(server.Close)

	client := http.Client{}
	response, err := client.Get(server.URL + "/capabilities")
	s.Require().NoError(err)

	s.Equal("application/json", response.Header["Content-Type"][0])

	body, err := io.ReadAll(response.Body)
	s.Require().NoError(err)

	spec := &CapabilitiesSpec{}
	err = json.Unmarshal(body, spec)
	s.Require().NoError(err)

	s.Len(spec.Endpoints, 6)
	s.Equal("/*", spec.Endpoints[0].Path)
	s.Equal("/capabilities", spec.Endpoints[1].Path)
	s.Equal("/reflect", spec.Endpoints[2].Path)
	s.Equal("/configure_reflection", spec.Endpoints[3].Path)
	s.Equal("/reset", spec.Endpoints[4].Path)
	s.Equal("/inspect", spec.Endpoints[5].Path)

	for _, ep := range spec.Endpoints {
		s.NotEmpty(ep.ContentType)
		s.NotEmpty(ep.Methods)
		s.NotEmpty(ep.Description)
	}
}

func (s *serverTestSuite) TestCapabilities_Quiet() {
	server := httptest.NewServer((http.HandlerFunc)(handleCapabilities))
	s.T().Cleanup(server.Close)

	client := http.Client{}
	response, err := client.Get(server.URL + "/capabilities/?quiet=true")
	s.Require().NoError(err)

	s.Equal("application/json", response.Header["Content-Type"][0])

	body, err := io.ReadAll(response.Body)
	s.Require().NoError(err)

	s.NotContains("methods", string(body))
	s.NotContains("contentType", string(body))
	s.NotContains("description", string(body))

	spec := &CapabilitiesSpec{}
	err = json.Unmarshal(body, spec)
	s.Require().NoError(err)

	s.Len(spec.Endpoints, 6)
	s.Equal("/*", spec.Endpoints[0].Path)
	s.Equal("/capabilities", spec.Endpoints[1].Path)
	s.Equal("/reflect", spec.Endpoints[2].Path)
	s.Equal("/configure_reflection", spec.Endpoints[3].Path)
	s.Equal("/reset", spec.Endpoints[4].Path)
	s.Equal("/inspect", spec.Endpoints[5].Path)
}

func (s *serverTestSuite) TestConfigureReflection() {
	server := httptest.NewServer((http.HandlerFunc)(handleConfigureReflection))
	s.T().Cleanup(server.Close)

	responseBodyString := "a dummy body \t \n\r\r\n\r\n"
	responseBody := base64.StdEncoding.EncodeToString([]byte(responseBodyString))
	spec := &configureReflectionSpec{
		reflectionSpec{
			Status: 202,
			Headers: map[string]string{
				"header1":  "value 1",
				"header_2": "value :2",
			},
			EncodedBody: responseBody,
		},
		[]dynamicEndpointSpec{
			{
				Method: "GET",
				Url:    "/foo/bar",
			},
			{
				Method: "POST",
				Url:    "/foo/bar?some=query",
			},
			{
				Method: "POST",
				Url:    "/foo?some=query",
			},
		},
	}
	body, err := json.Marshal(spec)
	s.Require().NoError(err)
	request, err := http.NewRequest("POST", server.URL+"/configure_reflection", bytes.NewReader(body))
	s.Require().NoError(err)
	client := http.Client{}
	_, err = client.Do(request)
	s.Require().NoError(err)

	defaultHandlingServer := httptest.NewServer((http.HandlerFunc)(handleDefault))
	s.T().Cleanup(defaultHandlingServer.Close)
	for _, _endpoint := range spec.Endpoints {
		request, err = http.NewRequest(_endpoint.Method, defaultHandlingServer.URL+_endpoint.Url, bytes.NewReader(body))
		s.Require().NoError(err)
		response, err := client.Do(request)
		s.Require().NoError(err)
		s.Equal(spec.Status, response.StatusCode)
		s.Len(response.Header, 5)

		s.Contains(response.Header, "Header1")
		s.Equal("value 1", response.Header["Header1"][0])
		s.Contains(response.Header, "Header_2")
		s.Equal("value :2", response.Header["Header_2"][0])

		reflectedBody, err := io.ReadAll(response.Body)
		s.Require().NoError(err)
		s.Equal(responseBodyString, string(reflectedBody))
	}
}

func (s *serverTestSuite) TestConfigureReflection_SameUrl() {
	server := httptest.NewServer((http.HandlerFunc)(handleConfigureReflection))
	s.T().Cleanup(server.Close)

	responseBodyString := "a dummy body \t \n\r\r\n\r\n"
	responseBody := base64.StdEncoding.EncodeToString([]byte(responseBodyString))
	spec := &configureReflectionSpec{
		reflectionSpec{
			Status: 202,
			Headers: map[string]string{
				"header1":  "value 1",
				"header_2": "value :2",
			},
			EncodedBody: responseBody,
		},
		[]dynamicEndpointSpec{
			{
				Method: "GET",
				Url:    "/foo/bar",
			},
			{
				Method: "OPTIONS",
				Url:    "/foo/bar",
			},
		},
	}
	body, err := json.Marshal(spec)
	s.Require().NoError(err)
	request, err := http.NewRequest("POST", server.URL+"/configure_reflection", bytes.NewReader(body))
	s.Require().NoError(err)
	client := http.Client{}
	_, err = client.Do(request)
	s.Require().NoError(err)

	defaultHandlingServer := httptest.NewServer((http.HandlerFunc)(handleDefault))
	s.T().Cleanup(defaultHandlingServer.Close)
	for _, _endpoint := range spec.Endpoints {
		request, err = http.NewRequest(_endpoint.Method, defaultHandlingServer.URL+_endpoint.Url, bytes.NewReader(body))
		s.Require().NoError(err)
		response, err := client.Do(request)
		s.Require().NoError(err)
		s.Equal(spec.Status, response.StatusCode)
		s.Len(response.Header, 5)

		s.Contains(response.Header, "Header1")
		s.Equal("value 1", response.Header["Header1"][0])
		s.Contains(response.Header, "Header_2")
		s.Equal("value :2", response.Header["Header_2"][0])

		reflectedBody, err := io.ReadAll(response.Body)
		s.Require().NoError(err)
		s.Equal(responseBodyString, string(reflectedBody))
	}
}

func (s *serverTestSuite) TestComputeEndPointKey() {
	key1 := computeEndpointKey("1", "2")
	key2 := computeEndpointKey("1", "2")
	s.Equal(key1, key2)
}

func (s *serverTestSuite) TestReset() {
	server := httptest.NewServer((http.HandlerFunc)(handleConfigureReflection))
	s.T().Cleanup(server.Close)

	spec := &configureReflectionSpec{
		reflectionSpec{
			Status:      234,
			Headers:     map[string]string{},
			EncodedBody: "",
		},
		[]dynamicEndpointSpec{
			{
				Method: "GET",
				Url:    "/foo/bar",
			},
		},
	}
	body, err := json.Marshal(spec)
	s.Require().NoError(err)
	request, err := http.NewRequest("POST", server.URL+"/configure_reflection", bytes.NewReader(body))
	s.Require().NoError(err)
	client := http.Client{}
	_, err = client.Do(request)
	s.Require().NoError(err)

	// check that the request is being reflected
	defaultHandlingServer := httptest.NewServer((http.HandlerFunc)(handleDefault))
	s.T().Cleanup(defaultHandlingServer.Close)
	request, err = http.NewRequest(spec.Endpoints[0].Method, defaultHandlingServer.URL+spec.Endpoints[0].Url, bytes.NewReader(body))
	s.Require().NoError(err)
	response, err := client.Do(request)
	s.Require().NoError(err)
	s.Equal(spec.Status, response.StatusCode)

	// reset
	resetHandlingServer := httptest.NewServer((http.HandlerFunc)(handleReset))
	s.T().Cleanup(resetHandlingServer.Close)
	request, err = http.NewRequest("PUT", resetHandlingServer.URL+"/reset", nil)
	s.Require().NoError(err)
	response, err = client.Do(request)
	s.Require().NoError(err)
	s.Equal(200, response.StatusCode)

	// check that now the default handler kicks in and no longer the configured reflection
	request, err = http.NewRequest(spec.Endpoints[0].Method, defaultHandlingServer.URL+spec.Endpoints[0].Url, bytes.NewReader(body))
	s.Require().NoError(err)
	response, err = client.Do(request)
	s.Require().NoError(err)
	s.Equal(200, response.StatusCode)
}
