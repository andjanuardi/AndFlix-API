package handler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"andflix/go-api/crypto"
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

func subtitleURL(raw interface{}) string {
	s, ok := raw.(string)
	if !ok || s == "" {
		if raw != nil {
			return fmt.Sprint(raw)
		}
		return ""
	}
	return "/subtitle?url=" + url.QueryEscape(s)
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

func callH5API(endpoint string, query map[string]string, aeskeyInternal, deviceID, token string) (int, map[string]interface{}, error) {
	params := make(map[string]interface{})
	for k, v := range query {
		params[k] = v
	}

	headers := crypto.H5BuildSignHeaders(params, deviceID, token)

	reqURL := h5APIBase + endpoint
	if len(query) > 0 {
		q := url.Values{}
		for k, v := range query {
			q.Set(k, v)
		}
		reqURL += "?" + q.Encode()
	}

	resp, err := forwardRequest("GET", reqURL, headers, nil)
	if err != nil {
		return 0, nil, fmt.Errorf("h5 request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("failed to read h5 response body: %w", err)
	}

	var respJSON map[string]interface{}
	if err := json.Unmarshal(respBody, &respJSON); err != nil {
		return resp.StatusCode, nil, fmt.Errorf("non-json response from %s: %s", endpoint, string(respBody))
	}

	return resp.StatusCode, respJSON, nil
}
