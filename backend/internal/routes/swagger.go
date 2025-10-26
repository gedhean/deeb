package routes

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/labstack/echo/v4"
)

// SetupSwaggerRoutes configures the Swagger UI routes
func SetupSwaggerRoutes(e *echo.Echo) {
	// Serve the OpenAPI specification file
	e.GET("/api/docs/openapi.yaml", func(c echo.Context) error {
		// Get the path to the openapi.yaml file
		specPath := filepath.Join("api", "openapi.yaml")

		// Check if file exists
		if _, err := os.Stat(specPath); os.IsNotExist(err) {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "OpenAPI specification not found",
			})
		}

		return c.File(specPath)
	})

	// Serve Swagger UI
	e.GET("/api/docs", func(c echo.Context) error {
		html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Deeb API Documentation</title>
    <link rel="stylesheet" type="text/css" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css">
    <style>
        body {
            margin: 0;
            padding: 0;
        }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            const ui = SwaggerUIBundle({
                url: "/api/docs/openapi.yaml",
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout",
                persistAuthorization: true
            });
            window.ui = ui;
        };
    </script>
</body>
</html>`
		return c.HTML(http.StatusOK, html)
	})

	// Redirect /api/docs/ to /api/docs
	e.GET("/api/docs/", func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, "/api/docs")
	})
}
