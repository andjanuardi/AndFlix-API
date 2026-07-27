package main

import (
	"fmt"
	"os"
	"time"

	"andflix/go-api/handler"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	port := "9996"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           86400 * time.Second,
	}))

	r.GET("/health", handler.HealthHandler)
	r.GET("/docs", handler.DocsHandler)
	r.GET("/openapi.yaml", handler.SpecHandler)

	r.GET("/image", handler.ImageProxy)
	r.GET("/subtitle", handler.SubtitleProxy)
	r.GET("/stream", handler.StreamProxy)

	r.POST("/getHome", handler.GetHomeCombined)

	r.POST("/search", handler.Search)

	r.POST("/getDetail", handler.GetDetail)

	fmt.Printf("ANDFLIX API running on http://0.0.0.0:%s\n", port)
	fmt.Printf("Endpoints:\n")
	fmt.Printf("  Core: /getHome (POST), /search (POST), /getDetail (POST)\n")
	fmt.Printf("  Image: /image?url=... (GET)\n")
	fmt.Printf("  Subtitle: /subtitle?url=... (GET)\n")
	fmt.Printf("  Stream: /stream?url=... (GET)\n")
	fmt.Printf("  Internal: /health, /docs, /openapi.yaml\n")

	if err := r.Run(":" + port); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start server: %v\n", err)
		os.Exit(1)
	}
}
