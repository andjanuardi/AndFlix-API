package handler

import (
	"io"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

func StreamProxy(c *gin.Context) {
	rawURL := c.Query("url")
	if rawURL == "" {
		c.JSON(400, gin.H{"error": "missing url parameter"})
		return
	}

	decoded, err := url.QueryUnescape(rawURL)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid url"})
		return
	}

	resp, err := httpClient.Get(decoded)
	if err != nil {
		c.JSON(502, gin.H{"error": "failed to fetch stream: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(502, gin.H{"error": "failed to read stream: " + err.Error()})
		return
	}

	contentType := resp.Header.Get("Content-Type")

	// M3U8 — rewrite URLs
	if strings.Contains(contentType, "mpegurl") || strings.Contains(contentType, "m3u8") || strings.HasSuffix(decoded, ".m3u8") {
		base := getBaseURL(decoded)
		lines := strings.Split(string(body), "\n")
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.HasPrefix(trimmed, "#") {
				continue
			}
			absURL := resolveURL(base, trimmed)
			lines[i] = "/stream?url=" + url.QueryEscape(absURL)
		}
		body = []byte(strings.Join(lines, "\n"))
		if contentType == "" {
			contentType = "application/vnd.apple.mpegurl"
		}
	}

	if contentType == "" {
		contentType = "application/octet-stream"
	}

	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "public, max-age=86400")
	c.Data(resp.StatusCode, contentType, body)
}
