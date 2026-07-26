package main

import (
	"fmt"
	"os"

	"andflix/go-api/handler"

	"github.com/gin-gonic/gin"
)

func main() {
	port := "8080"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.GET("/health", handler.HealthHandler)
	r.GET("/docs", handler.DocsHandler)
	r.GET("/openapi.yaml", handler.SpecHandler)

	r.GET("/image", handler.ImageProxy)
	r.GET("/subtitle", handler.SubtitleProxy)

	r.POST("/getHome", handler.GetHomeCombined)

	r.POST("/search", handler.Search)

	r.POST("/getDetail", handler.GetDetail)

	fmt.Printf("LOKLOK Proxy running on http://0.0.0.0:%s\n", port)
	fmt.Printf("Endpoints:\n")
	fmt.Printf("  Core: /getHome (POST), /search (POST), /getDetail (POST)\n")
	fmt.Printf("  Image: /image?url=... (GET)\n")
	fmt.Printf("  Subtitle: /subtitle?url=... (GET)\n")
	fmt.Printf("  Internal: /health, /docs, /openapi.yaml\n")

	if err := r.Run(":" + port); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start server: %v\n", err)
		os.Exit(1)
	}
}
