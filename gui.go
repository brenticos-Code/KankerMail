package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/webview/webview_go"
)

type NativeResponse struct {
	Status int    `json:"status"`
	Body   string `json:"body"`
}

// mockResponseWriter implements http.ResponseWriter for in-memory request routing
type mockResponseWriter struct {
	header http.Header
	body   strings.Builder
	status int
}

func (m *mockResponseWriter) Header() http.Header {
	return m.header
}

func (m *mockResponseWriter) Write(b []byte) (int, error) {
	return m.body.Write(b)
}

func (m *mockResponseWriter) WriteHeader(statusCode int) {
	m.status = statusCode
}

func startGui() {
	// Check for running graphical display server on Linux
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		fmt.Println("\n⚠️  Error: No graphical display server detected (DISPLAY/WAYLAND_DISPLAY is empty).")
		fmt.Println("To run the graphical client, you must run it from inside an active desktop session.")
		fmt.Println("Alternatively, you can run KankerMail in Web Server mode or Terminal TUI mode:")
		fmt.Println("  - Web UI:       ./kankermail -web")
		fmt.Println("  - Terminal TUI: ./kankermail")
		return
	}

	// Initialize global mail store with mock data
	globalMailStore = NewMailStore()

	w := webview.New(true)
	defer w.Destroy()

	w.SetTitle("KankerMail Desktop")
	w.SetSize(1100, 780, webview.HintNone)

	// Bind native JS request handler
	w.Bind("goRequest", func(method, path, body string) string {
		return handleNativeRequest(method, path, body)
	})

	// Inline embedded HTML, CSS, and JS
	inlinedHTML, err := getInlinedHTML()
	if err != nil {
		log.Fatalf("Failed to inline HTML assets: %v", err)
	}

	w.Navigate("data:text/html;charset=utf-8," + url.PathEscape(inlinedHTML))
	w.Run()
}

func getInlinedHTML() (string, error) {
	htmlBytes, err := webAssets.ReadFile("web/index.html")
	if err != nil {
		return "", err
	}
	htmlStr := string(htmlBytes)

	cssBytes, err := webAssets.ReadFile("web/style.css")
	if err != nil {
		return "", err
	}
	cssLinkTag := `<link rel="stylesheet" href="style.css">`
	htmlStr = strings.Replace(htmlStr, cssLinkTag, "<style>\n"+string(cssBytes)+"\n</style>", 1)

	jsBytes, err := webAssets.ReadFile("web/app.js")
	if err != nil {
		return "", err
	}
	jsScriptTag := `<script src="app.js"></script>`
	htmlStr = strings.Replace(htmlStr, jsScriptTag, "<script>\n"+string(jsBytes)+"\n</script>", 1)

	return htmlStr, nil
}

func handleNativeRequest(method, path, body string) string {
	// Parse URL properly for request mock
	req, err := http.NewRequest(method, path, strings.NewReader(body))
	if err != nil {
		return fmt.Sprintf("Error creating request: %v", err)
	}

	w := &mockResponseWriter{
		header: make(http.Header),
		status: 200,
	}

	// Parse query parameters
	parsedURL, err := url.Parse(path)
	if err == nil {
		req.URL = parsedURL
	}

	// Route based on URL path prefix
	cleanPath := req.URL.Path
	if cleanPath == "/api/config" {
		if method == http.MethodGet {
			handleGetConfig(w, req)
		} else {
			handlePostConfig(w, req)
		}
	} else if cleanPath == "/api/folders" {
		handleGetFolders(w, req)
	} else if cleanPath == "/api/emails" {
		handleGetEmails(w, req)
	} else if cleanPath == "/api/email" {
		handleGetEmail(w, req)
	} else if cleanPath == "/api/action" {
		handleAction(w, req)
	} else if cleanPath == "/api/send" {
		handleSend(w, req)
	} else {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	}

	log.Printf("[Native TUI Bridge] %s %s -> Status: %d, Body Size: %d bytes", method, path, w.status, w.body.Len())

	resp := NativeResponse{
		Status: w.status,
		Body:   w.body.String(),
	}
	respJSON, err := json.Marshal(resp)
	if err != nil {
		return fmt.Sprintf(`{"status":500,"body":"JSON marshal error: %s"}`, err.Error())
	}
	return string(respJSON)
}
