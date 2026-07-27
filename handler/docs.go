package handler

import "github.com/gin-gonic/gin"

func DocsHandler(c *gin.Context) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8" />
<title>AndFlix — ANDFLIX API Proxy</title>
<link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
<script>
SwaggerUIBundle({
  url: "/openapi.yaml",
  dom_id: "#swagger-ui",
  presets: [SwaggerUIBundle.presets.apis],
  layout: "BaseLayout",
  deepLinking: true
});
</script>
</body>
</html>`
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(200, html)
}
