# Albedo - HTTP reflector and black hole

[![Go Report Card](https://goreportcard.com/badge/github.com/coreruleset/albedo)](https://goreportcard.com/report/github.com/coreruleset/albedo)
[![Release](https://img.shields.io/github/v/release/coreruleset/albedo.svg?style=flat-square)](https://github.com/coreruleset/albedo/releases/latest)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/coreruleset/albedo/badge)](https://securityscorecards.dev/viewer/?uri=github.com/coreruleset/albedo)

Albedo is a simple HTTP server used as a reverse-proxy backend in testing web application firewalls (WAFs). [go-ftw](https://github.com/coreruleset/go-ftw) relies on Albedo to test WAF rules of responses.

## Usage

```bash
$ albedo -h
HTTP reflector and black hole

Usage:
  albedo [flags]

Flags:
  -b, --bind string   address to bind to (default "0.0.0.0")
  -h, --help          help for albedo
  -p, --port int      port to listen on (default 8080)
      --debug         log debug information
```

## Usage as a library
`github.com/coreruleset/albedo/server` package provides a handler that can be used for testing purposes.
Usage example:
```go
package albedo_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coreruleset/albedo/server"
	"github.com/stretchr/testify/require"
)

func TestAlbedo(t *testing.T) {
	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	client := http.Client{
		Timeout: time.Duration(1 * time.Second),
	}

	_, err := client.Get(testServer.URL)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}
```

## Endpoints

```yaml
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
          body        [string]: body of the response
          encodedBody [base64-encoded string]: body of the response, base64-encoded; useful for complex payloads where escaping is difficult
          logMessage  [string]: message to log for the request; useful for matching requests to tests

      While this endpoint essentially allows for freeform responses, some restrictions apply:
        - responses with status code 1xx don't have a body; if you specify a body together with a 1xx
          status code, the behavior is undefined
        - response status codes must lie in the range of [100,599]
        - the names and values of headers must be valid according to the HTTP specification;
          invalid headers will be dropped
```
