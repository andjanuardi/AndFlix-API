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

	r.POST("/getHome", handler.GetHomeCombined)

	r.POST("/search", handler.Search)

	r.GET("/cms/web/movieDrama/get", handler.H5MovieDramaGet)
	r.POST("/cms/web/movieDrama/get", handler.H5MovieDramaGet)
	r.PUT("/cms/web/movieDrama/get", handler.H5MovieDramaGet)
	r.DELETE("/cms/web/movieDrama/get", handler.H5MovieDramaGet)
	r.PATCH("/cms/web/movieDrama/get", handler.H5MovieDramaGet)
	r.OPTIONS("/cms/web/movieDrama/get", handler.H5MovieDramaGet)

	r.GET("/cms/web/ios_h5/movieDrama/getPlayInfo", handler.H5GetPlayInfo)
	r.POST("/cms/web/ios_h5/movieDrama/getPlayInfo", handler.H5GetPlayInfo)
	r.PUT("/cms/web/ios_h5/movieDrama/getPlayInfo", handler.H5GetPlayInfo)
	r.DELETE("/cms/web/ios_h5/movieDrama/getPlayInfo", handler.H5GetPlayInfo)
	r.PATCH("/cms/web/ios_h5/movieDrama/getPlayInfo", handler.H5GetPlayInfo)
	r.OPTIONS("/cms/web/ios_h5/movieDrama/getPlayInfo", handler.H5GetPlayInfo)

	fmt.Printf("LOKLOK Proxy running on http://0.0.0.0:%s\n", port)
	fmt.Printf("Endpoints:\n")
	fmt.Printf("  PC: /search (POST), /getHome (POST)\n")
	fmt.Printf("  H5: /cms/web/movieDrama/get, /cms/web/ios_h5/movieDrama/getPlayInfo\n")
	fmt.Printf("  Image: /image?url=... (GET)\n")
	fmt.Printf("  Internal: /health, /docs, /openapi.yaml\n")

	if err := r.Run(":" + port); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start server: %v\n", err)
		os.Exit(1)
	}
}
