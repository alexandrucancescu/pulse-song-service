package poster

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"pulse-song-service/config"
)

var client = &http.Client{
	Timeout: 10 * time.Second,
}

// PostToAll sends the content to all configured endpoints.
// Logs success/failure for each endpoint and a summary at the end.
func PostToAll(endpoints []config.Endpoint, content string) {
	succeeded := 0

	for _, ep := range endpoints {
		if err := postToEndpoint(ep, content); err != nil {
			log.Printf("ERROR: %s — %v", ep.URL, err)
		} else {
			succeeded++
		}
	}

	log.Printf("%d/%d endpoints called successfully", succeeded, len(endpoints))
}

// postToEndpoint sends a POST request with JSON body {postKey: content} to one endpoint.
func postToEndpoint(ep config.Endpoint, content string) error {
	body := map[string]string{ep.PostKey: content}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("cannot encode JSON: %w", err)
	}

	req, err := http.NewRequest("POST", ep.URL, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("cannot create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for key, value := range ep.Headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	// Read the error response body for debugging.
	respBody, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("status %d: %s", resp.StatusCode, string(respBody))
}
