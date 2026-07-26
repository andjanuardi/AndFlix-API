package handler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

var deviceKeyCache sync.Map

const (
	pcAPIBase = "https://api.loklok.fun"
	h5APIBase = "https://h5-api.hehekang.com"
)

var httpClient = &http.Client{
	Timeout: 60 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return fmt.Errorf("too many redirects")
		}
		return nil
	},
}

func extractDeviceIDFromToken(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return ""
	}
	payloadB64 := parts[1]
	switch len(payloadB64) % 4 {
	case 2:
		payloadB64 += "=="
	case 3:
		payloadB64 += "="
	}
	payloadBytes, err := base64.URLEncoding.DecodeString(payloadB64)
	if err != nil {
		return ""
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return ""
	}
	deviceID, _ := payload["deviceId"].(string)
	return deviceID
}

func forwardRequest(method, url string, headers map[string]string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return httpClient.Do(req)
}
