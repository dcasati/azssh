package azssh

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

func sendRequest(token string, method string, url string, payload string) (map[string]interface{}, error) {
	client := &http.Client{}

	req, err := http.NewRequest(method, url, bytes.NewBuffer([]byte(payload)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Add("Referer", "https://portal.azure.com/")
	req.Header.Add("x-ms-console-preferred-location", "westus")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}
	return result, nil
}

func createConsole(token string) (string, error) {
	fmt.Println("Requesting a Cloud Shell.")
	data := `{"properties": { "osType": "linux" } }`
	result, err := sendRequest(token, "PUT", "https://management.azure.com/providers/Microsoft.Portal/consoles/default?api-version=2023-02-01-preview", data)
	if err != nil {
		return "", fmt.Errorf("failed to create console: %w", err)
	}
	
	properties, ok := result["properties"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid response: missing or invalid 'properties' field")
	}
	
	uri, ok := properties["uri"].(string)
	if !ok || uri == "" {
		return "", fmt.Errorf("invalid response: missing or invalid 'uri' field")
	}
	
	return uri, nil
}

func createTerminal(token string, consoleURL string, shellType string, initialSize TerminalSize) (string, string, string, error) {
	fmt.Println("Connecting terminal...")
	url := fmt.Sprintf("%s/terminals?cols=%d&rows=%d&shell=%s", consoleURL, initialSize.Cols, initialSize.Rows, shellType)
	data := `{"tokens": []}`
	
	// Retry logic: Azure Cloud Shell may need time to start the container
	maxRetries := 5
	retryDelay := 3 * time.Second
	
	var result map[string]interface{}
	var err error
	
	for attempt := 1; attempt <= maxRetries; attempt++ {
		result, err = sendRequest(token, "POST", url, data)
		if err == nil {
			break
		}
		
		// Check if it's a 404 with "EndpointNotFound" or "no listeners connected"
		if strings.Contains(err.Error(), "404") && 
		   (strings.Contains(err.Error(), "EndpointNotFound") || 
		    strings.Contains(err.Error(), "no listeners")) {
			if attempt < maxRetries {
				fmt.Printf("Cloud Shell container is starting... (attempt %d/%d, waiting %v)\n", attempt, maxRetries, retryDelay)
				time.Sleep(retryDelay)
				continue
			}
		}
		
		// For other errors or if we've exhausted retries, return the error
		return "", "", "", fmt.Errorf("failed to create terminal: %w", err)
	}
	
	id, ok := result["id"].(string)
	if !ok || id == "" {
		return "", "", "", fmt.Errorf("invalid response: missing or invalid 'id' field")
	}
	
	socketUri, ok := result["socketUri"].(string)
	if !ok || socketUri == "" {
		return "", "", "", fmt.Errorf("invalid response: missing or invalid 'socketUri' field")
	}
	
	// Logic based on Microsoft Terminal's AzureConnection.cpp
	// https://github.com/microsoft/terminal/blob/main/src/cascadia/TerminalConnection/AzureConnection.cpp
	var finalSocketUri string
	
	if !strings.Contains(consoleURL, "servicebus") {
		// Simple case: cloudShellUri doesn't contain "servicebus"
		// Remove "https" from cloudShellUri and construct wss URL
		uriWithoutProtocol := strings.TrimPrefix(consoleURL, "https")
		finalSocketUri = fmt.Sprintf("wss%s/terminals/%s", uriWithoutProtocol, id)
	} else {
		// Complex case: servicebus URLs need $hc path
		// Convert https://host.servicebus.windows.net:443/cc-AAAA-AAAAAAAA
		// to wss://host.servicebus.windows.net:443/$hc/cc-AAAA-AAAAAAAA/terminals/sessionId
		
		// Replace https with wss
		wsURL := strings.Replace(consoleURL, "https://", "wss://", 1)
		
		// Find the path part (after host:port)
		// URL format: wss://host:port/path
		slashAfterHost := strings.Index(wsURL[6:], "/") // Skip "wss://"
		if slashAfterHost >= 0 {
			// Insert /$hc before the path
			slashAfterHost += 6 // adjust for the "wss://" we skipped
			finalSocketUri = fmt.Sprintf("%s/$hc%s/terminals/%s", wsURL[:slashAfterHost], wsURL[slashAfterHost:], id)
		} else {
			// No path after host
			finalSocketUri = fmt.Sprintf("%s/$hc/terminals/%s", wsURL, id)
		}
	}
	
	return id, finalSocketUri, consoleURL, nil
}

func resizeTerminal(token string, consoleURL string, terminalID string, resize <-chan TerminalSize) {
	for {
		newSize := <-resize
		url := fmt.Sprintf("%s/terminals/%s/size?cols=%d&rows=%d", consoleURL, terminalID, newSize.Cols, newSize.Rows)
		if _, err := sendRequest(token, "POST", url, ""); err != nil {
			log.Println("Failed to resize terminal:", err)
		}
	}
}

// ProvisionCloudShell sets up a Cloud Shell and a websocket to connect into it
func ProvisionCloudShell(token string, shellType string, initialSize TerminalSize, resize <-chan TerminalSize) (string, string, error) {
	consoleURL, err := createConsole(token)
	if err != nil {
		return "", "", err
	}
	
	terminalID, websocketURI, _, err := createTerminal(token, consoleURL, shellType, initialSize)
	if err != nil {
		return "", "", err
	}
	
	go resizeTerminal(token, consoleURL, terminalID, resize)
	return websocketURI, token, nil
}
