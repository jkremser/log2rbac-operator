package internal

import (
	"fmt"
	"io"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"strings"
)

var httpLog = ctrl.Log.WithName("http-server")

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
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := io.WriteString(w, content)
		if err != nil {
			httpLog.Error(err, "Error when querying /")
		}
	}
}

func ServeRoot(mgr manager.Manager, cfg AppConfig) error {
	content := getContent(cfg)
	return mgr.AddMetricsExtraHandler("/", http.HandlerFunc(getServeHTTPFunc(content)))
}
