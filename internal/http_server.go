package internal

import (
	"fmt"
	"io"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"strings"
)

func sanitize(i string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(i, "<", ""), ">", ""), "&", "")
}

func getContent(cfg AppConfig) string {
	// injection sanitizing
	version := sanitize(cfg.Version)
	sha := sanitize(cfg.GitSha)
	bodyContent := fmt.Sprintf(`
<h1>log2rbac-operator</h1>
<br/><br/>
<br/><b>version:</b> %s
<br/><b>git sha:</b> %s
<br/><b>metrics:</b> <a href="/metrics">/metrics</a>`, version, sha)
	return fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
	<head>
		<meta charset="utf-8">
		<title>log2rbac</title>
	</head>
	<body>
		<div style="text-align: center;;font-family: Arial, Helvetica, sans-serif;">
			%s
		</div>
	</body>
</html>`, bodyContent)
}

func getServeHTTPFunc(content string) func(http.ResponseWriter, *http.Request) {
	return func (w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, content)
	}
}

func ServeRoot(mgr manager.Manager, cfg AppConfig) error {
	content := getContent(cfg)
	return mgr.AddMetricsExtraHandler("/", http.HandlerFunc(getServeHTTPFunc(content)))
}
