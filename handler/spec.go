package handler

import (
	"os"

	"github.com/gin-gonic/gin"
)

func SpecHandler(c *gin.Context) {
	data, err := os.ReadFile("openapi.yaml")
	if err != nil {
		c.JSON(404, gin.H{"error": "openapi.yaml not found"})
		return
	}
	c.Header("Content-Type", "text/yaml; charset=utf-8")
	c.String(200, string(data))
}
