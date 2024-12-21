package machinestats

import (
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartHTTPServer(t *testing.T) {
	port := 8081 // Choose a port that is likely to be available

	// Start the server
	mux, stop, err := StartHTTPServer(port)
	require.NoError(t, err, "Expected no error when starting the server")
	require.NotNil(t, mux, "Expected a valid ServeMux")
	require.NotNil(t, stop, "Expected a valid stop function")

	// Ensure server is running by making a request
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/", port))
	require.NoError(t, err, "Expected no error when making a GET request")
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Expected no error when reading response body")
	assert.Equal(t, "OK\n", string(body), "Expected response body to be 'OK'")

	// Stop the server
	err = stop()
	assert.NoError(t, err, "Expected no error when stopping the server")

	// Ensure server has stopped by making a request
	_, err = http.Get(fmt.Sprintf("http://localhost:%d/", port))
	assert.Error(t, err, "Expected an error when making a GET request after server is stopped")
}

func TestStartHTTPServer_PortInUse(t *testing.T) {
	port := 8082 // Choose a port that is likely to be available

	// Start the first server
	_, stop1, err := StartHTTPServer(port)
	require.NoError(t, err, "Expected no error when starting the first server")
	defer stop1()

	// Attempt to start a second server on the same port
	_, _, err = StartHTTPServer(port)
	assert.Error(t, err, "Expected an error when starting a second server on the same port")
}
