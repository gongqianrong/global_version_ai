package api

import (
	"embed"
	"net/http"
)

//go:embed swagger_res/*
var swaggerRes embed.FS

const swaggerHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Rakutao API Documentation</title>
  <link rel="stylesheet" href="./swagger-ui.min.css">
  <style>
    html { box-sizing: border-box; overflow-y: scroll; }
    *, *:before, *:after { box-sizing: inherit; }
    body { margin: 0; background: #fafafa; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="./swagger-ui-bundle.min.js"></script>
  <script>
    SwaggerUIBundle({
      url: './openapi.yaml',
      dom_id: '#swagger-ui',
      deepLinking: true,
      presets: [
        SwaggerUIBundle.presets.apis,
        SwaggerUIBundle.SwaggerUIStandalonePreset,
      ],
      layout: 'BaseLayout',
    });
  </script>
</body>
</html>`

// HandleSwaggerUI serves the Swagger UI HTML page.
func HandleSwaggerUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(swaggerHTML))
}

func serveEmbedded(name, contentType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := swaggerRes.ReadFile("swagger_res/" + name)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.Write(data)
	}
}

// HandleOpenAPISpec serves the OpenAPI YAML spec.
var HandleOpenAPISpec = serveEmbedded("openapi.yaml", "application/yaml")

// HandleSwaggerJS serves the embedded swagger-ui-bundle JS.
var HandleSwaggerJS = serveEmbedded("swagger-ui-bundle.min.js", "application/javascript")

// HandleSwaggerCSS serves the embedded swagger-ui CSS.
var HandleSwaggerCSS = serveEmbedded("swagger-ui.min.css", "text/css")
