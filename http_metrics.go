package machinestats

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type HTTPStat struct {
	url    string
	name   string
	prefix string
}

func NewHTTPStat(name string, url string, prefix string) *HTTPStat {
	return &HTTPStat{
		name:   name,
		url:    url,
		prefix: prefix,
	}
}
func (h *HTTPStat) Name() string {
	return h.name
}

func (h *HTTPStat) Measure(channel chan<- Measurement) error {
	data, err := FetchAndFlattenJSON(h.url)
	if err != nil {
		return err
	}
	for k, v := range data {
		prefixedName := k
		if h.prefix != "" {
			prefixedName = fmt.Sprintf("%v.%v", h.prefix, k)
		}
		m := &BasicMeasurement{
			name:            prefixedName,
			measurementType: Gauge,
			value:           v,
		}
		channel <- m
	}
	return nil
}

// FetchAndFlattenJSON makes a GET request to the given URL and returns a flattened JSON map.
func FetchAndFlattenJSON(url string) (map[string]interface{}, error) {
	// Make the GET request
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to make GET request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	// Parse the JSON response
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	// Flatten the JSON map
	flattenedData := flattenMap(response, "")

	return flattenedData, nil
}

// flattenMap flattens a nested map by appending sub-keys with a '.' separator.
func flattenMap(data map[string]interface{}, prefix string) map[string]interface{} {
	flatMap := make(map[string]interface{})

	for key, value := range data {
		fullKey := key
		if prefix != "" {
			fullKey = fmt.Sprintf("%v.%v", prefix, key)
		}

		switch nestedValue := value.(type) {
		case map[string]interface{}:
			// Recursively flatten nested maps
			nestedFlatMap := flattenMap(nestedValue, fullKey)
			for nestedKey, nestedVal := range nestedFlatMap {
				flatMap[nestedKey] = nestedVal
			}
		default:
			flatMap[fullKey] = value
		}
	}

	return flatMap
}
