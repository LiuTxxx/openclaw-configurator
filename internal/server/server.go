package server

import (
	"net"
	"net/http"
	"strconv"

	"github.com/teecert/openclaw-configurator/web"
)

func NewServer(token string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/connect", handleConnect)
	mux.HandleFunc("/api/disconnect", handleDisconnect)
	mux.HandleFunc("/api/detect", handleDetect)
	mux.HandleFunc("/api/path", handleSetPath)
	mux.HandleFunc("/api/models", handleGetModels)
	mux.HandleFunc("/api/agents", handleGetAgents)
	mux.HandleFunc("/api/providers", handleProviders)
	mux.HandleFunc("/api/providers/", handleProviders)
	mux.HandleFunc("/api/primary", handleSetPrimary)
	mux.HandleFunc("/api/save", handleSave)
	mux.HandleFunc("/api/rawconfig", handleRawConfig)
	mux.HandleFunc("/api/test", handleTestProvider)
	mux.HandleFunc("/api/export", handleExport)
	mux.HandleFunc("/api/import", handleImport)
	mux.HandleFunc("/api/diff", handleDiff)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data, err := web.GetIndex()
		if err != nil {
			http.Error(w, "internal error", 500)
			return
		}
		w.Write(data)
	})

	var handler http.Handler = mux
	handler = maxBodySize(handler)
	handler = serializeRequests(handler)
	handler = authMiddleware(token, handler)
	handler = securityHeaders(handler)

	return handler
}

func ListenAndServe(bind string, port int, token string) error {
	handler := NewServer(token)
	addr := net.JoinHostPort(bind, strconv.Itoa(port))
	return http.ListenAndServe(addr, handler)
}
