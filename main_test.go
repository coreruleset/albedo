package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coreruleset/albedo/server"
	"github.com/stretchr/testify/require"
)

func TestAlbedoLibrary(t *testing.T) {
	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	client := http.Client{
		Timeout: time.Duration(1 * time.Second),
	}

	resp, err := client.Get(testServer.URL)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}
