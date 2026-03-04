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
  <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/4.15.5/swagger-ui.min.css">
  <style>
    html { box-sizing: border-box; overflow-y: scroll; }
    *, *:before, *:after { box-sizing: inherit; }
    body { margin: 0; background: #fafafa; }
    .api-header {
      background: #1b1b1b; color: #fff; padding: 12px 24px;
      display: flex; align-items: center; gap: 12px; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
    }
    .api-header h1 { margin: 0; font-size: 20px; font-weight: 600; }
    .version-badge {
      background: #22c55e; color: #fff; padding: 2px 10px; border-radius: 12px;
      font-size: 13px; font-weight: 600;
    }
    .new-badge {
      background: #ef4444; color: #fff; padding: 2px 8px; border-radius: 8px;
      font-size: 11px; font-weight: 700; margin-left: 4px; animation: pulse 1.5s infinite;
    }
    @keyframes pulse { 0%,100% { opacity: 1; } 50% { opacity: 0.5; } }
    .changelog-btn {
      margin-left: auto; background: transparent; border: 1px solid #555; color: #ccc;
      padding: 6px 16px; border-radius: 6px; cursor: pointer; font-size: 13px; transition: all .2s;
    }
    .changelog-btn:hover { border-color: #888; color: #fff; }
    .changelog-panel {
      display: none; max-width: 960px; margin: 0 auto; padding: 20px 24px;
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
    }
    .changelog-panel.open { display: block; }
    .changelog-panel h2 { font-size: 18px; margin: 0 0 16px; color: #333; }
    .version-section { margin-bottom: 20px; border: 1px solid #e5e7eb; border-radius: 8px; overflow: hidden; }
    .version-header { background: #f9fafb; padding: 12px 16px; border-bottom: 1px solid #e5e7eb; }
    .version-header strong { font-size: 15px; }
    .version-header .date { color: #6b7280; font-size: 13px; margin-left: 8px; }
    .version-header .summary { color: #6b7280; font-size: 13px; display: block; margin-top: 4px; }
    .change-list { padding: 12px 16px; margin: 0; }
    .change-item { display: flex; gap: 8px; margin-bottom: 10px; font-size: 14px; line-height: 1.5; }
    .change-item:last-child { margin-bottom: 0; }
    .change-tag {
      display: inline-block; padding: 1px 8px; border-radius: 4px; font-size: 11px;
      font-weight: 700; text-transform: uppercase; flex-shrink: 0; height: 20px; line-height: 18px; margin-top: 2px;
    }
    .tag-added { background: #dcfce7; color: #166534; }
    .tag-modified { background: #fef9c3; color: #854d0e; }
    .tag-fixed { background: #dbeafe; color: #1e40af; }
    .tag-deprecated { background: #f3e8ff; color: #6b21a8; }
    .tag-removed { background: #fee2e2; color: #991b1b; }
    .change-endpoints { color: #6b7280; font-size: 12px; font-family: monospace; }
  </style>
</head>
<body>
  <div class="api-header">
    <h1>Rakutao API</h1>
    <span class="version-badge" id="versionBadge"></span>
    <span class="new-badge" id="newBadge" style="display:none">NEW</span>
    <button class="changelog-btn" id="changelogBtn" onclick="toggleChangelog()">Changelog</button>
  </div>
  <div class="changelog-panel" id="changelogPanel"></div>
  <div id="swagger-ui"></div>
  <script src="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/4.15.5/swagger-ui-bundle.min.js"></script>
  <script>
    SwaggerUIBundle({
      url: './openapi.yaml',
      dom_id: '#swagger-ui',
      deepLinking: true,
      presets: [SwaggerUIBundle.presets.apis],
      layout: 'BaseLayout',
    });

    function toggleChangelog() {
      var p = document.getElementById('changelogPanel');
      var open = p.classList.toggle('open');
      document.getElementById('changelogBtn').textContent = open ? 'Hide Changelog' : 'Changelog';
    }

    fetch('./changelog.json').then(function(r){return r.json()}).then(function(data){
      var versions = data.versions || [];
      if (!versions.length) return;
      var latest = versions[0];
      document.getElementById('versionBadge').textContent = 'v' + latest.version;
      var d = new Date(latest.date);
      var now = new Date();
      if ((now - d) / 86400000 < 7) {
        document.getElementById('newBadge').style.display = 'inline';
      }
      var tagClass = {added:'tag-added',modified:'tag-modified',fixed:'tag-fixed',deprecated:'tag-deprecated',removed:'tag-removed'};
      var html = '<h2>Changelog</h2>';
      versions.forEach(function(v){
        html += '<div class="version-section"><div class="version-header"><strong>v'+v.version+'</strong><span class="date">'+v.date+'</span>';
        if(v.summary) html += '<span class="summary">'+v.summary+'</span>';
        html += '</div><div class="change-list">';
        (v.changes||[]).forEach(function(c){
          var cls = tagClass[c.type]||'tag-added';
          html += '<div class="change-item"><span class="change-tag '+cls+'">'+c.type+'</span><div>'+c.description;
          if(c.endpoints&&c.endpoints.length){
            html += '<div class="change-endpoints">'+c.endpoints.join(' &middot; ')+'</div>';
          }
          html += '</div></div>';
        });
        html += '</div></div>';
      });
      document.getElementById('changelogPanel').innerHTML = html;
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

// HandleChangelogJSON serves the changelog JSON file.
var HandleChangelogJSON = serveEmbedded("changelog.json", "application/json")

// HandleSwaggerJS serves the embedded swagger-ui-bundle JS.
var HandleSwaggerJS = serveEmbedded("swagger-ui-bundle.min.js", "application/javascript")

// HandleSwaggerCSS serves the embedded swagger-ui CSS.
var HandleSwaggerCSS = serveEmbedded("swagger-ui.min.css", "text/css")
