package handler

import (
	"io"
	"net/url"

	"github.com/gin-gonic/gin"
)

func SubtitleProxy(c *gin.Context) {
	subURL := c.Query("url")
	if subURL == "" {
		c.JSON(400, gin.H{"error": "missing url parameter"})
		return
	}

	decoded, err := url.QueryUnescape(subURL)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid url"})
		return
	}

	resp, err := httpClient.Get(decoded)
	if err != nil {
		c.JSON(502, gin.H{"error": "failed to fetch subtitle: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(502, gin.H{"error": "failed to read subtitle: " + err.Error()})
		return
	}

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Cache-Control", "public, max-age=86400")
	c.Data(resp.StatusCode, "text/plain; charset=utf-8", body)
}
