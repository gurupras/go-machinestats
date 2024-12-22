package machinestats

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFetchAndFlattenJSON(t *testing.T) {
	// Mock server to simulate HTTP responses
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"key1": "value1",
			"key2": map[string]interface{}{
				"subkey1": "subvalue1",
				"subkey2": "subvalue2",
			},
			"key3": 42,
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	// Test the FetchAndFlattenJSON function
	expected := map[string]interface{}{
		"key1":         "value1",
		"key2.subkey1": "subvalue1",
		"key2.subkey2": "subvalue2",
		"key3":         float64(42), // JSON numbers are float64 by default
	}

	result, err := FetchAndFlattenJSON(mockServer.URL)
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestFetchAndFlattenJSON_Error(t *testing.T) {
	// Mock server to simulate an error response
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer mockServer.Close()

	// Test the FetchAndFlattenJSON function for error handling
	result, err := FetchAndFlattenJSON(mockServer.URL)
	assert.Error(t, err)
	assert.Nil(t, result)
}

// Mock HTTP client to simulate network errors
var mockHTTPClient = &http.Client{
	Transport: &mockTransport{},
}

type mockTransport struct{}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Simulate a network error
	return nil, errors.New("mock network error")
}

func TestFetchAndFlattenJSON_NetworkError(t *testing.T) {
	// Temporarily replace the default HTTP client
	oldClient := http.DefaultClient
	http.DefaultClient = mockHTTPClient
	defer func() { http.DefaultClient = oldClient }()

	// Test for network error
	result, err := FetchAndFlattenJSON("http://invalid.url")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "mock network error")
}

func TestFetchAndFlattenJSON_HTTPError(t *testing.T) {
	// Mock server to simulate an HTTP error response
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer mockServer.Close()

	// Test for HTTP error response
	result, err := FetchAndFlattenJSON(mockServer.URL)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestFetchAndFlattenJSON_InvalidJSON(t *testing.T) {
	// Mock server to simulate an invalid JSON response
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("invalid JSON"))
	}))
	defer mockServer.Close()

	// Test for invalid JSON
	result, err := FetchAndFlattenJSON(mockServer.URL)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to unmarshal JSON")
}

func TestFlattenMap(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "Simple map with no nesting",
			input: map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
			expected: map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name: "Nested map",
			input: map[string]interface{}{
				"key1": "value1",
				"key2": map[string]interface{}{
					"subkey1": "subvalue1",
					"subkey2": "subvalue2",
				},
				"key3": 42,
			},
			expected: map[string]interface{}{
				"key1":         "value1",
				"key2.subkey1": "subvalue1",
				"key2.subkey2": "subvalue2",
				"key3":         42,
			},
		},
		{
			name: "Deeply nested map",
			input: map[string]interface{}{
				"key1": map[string]interface{}{
					"subkey1": map[string]interface{}{
						"subsubkey1": "subsubvalue1",
					},
				},
			},
			expected: map[string]interface{}{
				"key1.subkey1.subsubkey1": "subsubvalue1",
			},
		},
		{
			name:     "Empty map",
			input:    map[string]interface{}{},
			expected: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := flattenMap(tt.input, "")
			assert.Equal(t, tt.expected, result)
		})
	}
}
