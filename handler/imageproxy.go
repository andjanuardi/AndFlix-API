package handler

import (
	"io"

	"github.com/gin-gonic/gin"
)

func ImageProxy(c *gin.Context) {
	imageURL := c.Query("url")
	if imageURL == "" {
		c.JSON(400, gin.H{"error": "missing url parameter"})
		return
	}

	resp, err := httpClient.Get(imageURL)
	if err != nil {
		c.JSON(502, gin.H{"error": "failed to fetch image: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(502, gin.H{"error": "failed to read image: " + err.Error()})
		return
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}

	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "public, max-age=86400")
	c.Data(resp.StatusCode, contentType, body)
}
