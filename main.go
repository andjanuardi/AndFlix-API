package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"andflix/go-api/handler"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--install" {
		install()
		return
	}

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

func install() {
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get executable path: %v\n", err)
		os.Exit(1)
	}

	dest := "/usr/local/bin/andflix-api"
	port := "9996"
	svc := "/etc/systemd/system/andflix-api.service"

	data, err := os.ReadFile(exe)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read binary: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(dest, data, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to copy binary: %v\n", err)
		os.Exit(1)
	}

	content := fmt.Sprintf(`[Unit]
Description=ANDFLIX API Service
After=network.target

[Service]
Type=simple
User=root
ExecStart=%s %s
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
`, dest, port)

	if err := os.WriteFile(svc, []byte(content), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write service file: %v\n", err)
		os.Exit(1)
	}

	exec.Command("systemctl", "daemon-reload").Run()
	exec.Command("systemctl", "enable", "andflix-api").Run()
	exec.Command("systemctl", "restart", "andflix-api").Run()

	fmt.Println("ANDFLIX API Service installed and running on port", port)
}
