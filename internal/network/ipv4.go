package network

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func GetGlobalIPv4() (string, error) {
	var client = &http.Client{
		Timeout: 5 * time.Second,
	}

	body, err := ipify(client)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return parseIP(body)
}

func ipify(client *http.Client) ([]byte, error) {
	resp, err := client.Get("https://api.ipify.org?format=json")
	if err != nil {
		return nil, fmt.Errorf("failed to get global IPv4 address: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func parseIP(body []byte) (string, error) {
	var result struct {
		IP string `json:"ip"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse JSON: %w", err)
	}

	if result.IP == "" {
		return "", fmt.Errorf("no IP found in response")
	}

	return result.IP, nil
}
