package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

//go:embed web/*
var webAssets embed.FS

var (
	globalMailStore *MailStore
	mailStoreMutex  sync.Mutex
)

func startWebServer(port int) {
	// Initialize global mail store with mock data
	globalMailStore = NewMailStore()

	// Serve static web assets
	staticFS, err := fs.Sub(webAssets, "web")
	if err != nil {
		log.Fatalf("Failed to initialize static file system: %v", err)
	}
	http.Handle("/", http.FileServer(http.FS(staticFS)))

	// API Handlers
	http.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handleGetConfig(w, r)
		} else if r.Method == http.MethodPost {
			handlePostConfig(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/api/folders", handleGetFolders)
	http.HandleFunc("/api/emails", handleGetEmails)
	http.HandleFunc("/api/email", handleGetEmail)
	http.HandleFunc("/api/action", handleAction)
	http.HandleFunc("/api/send", handleSend)

	fmt.Printf("⚡ KankerMail Web Client running at http://localhost:%d\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

func handleGetConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := LoadConfig()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfg)
}

func handlePostConfig(w http.ResponseWriter, r *http.Request) {
	var update map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		log.Printf("ERROR: handlePostConfig failed to decode JSON: %v", err)
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	cfg, err := LoadConfig()
	if err != nil {
		log.Printf("ERROR: LoadConfig failed: %v", err)
		http.Error(w, fmt.Sprintf("Failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	// Update fields dynamically
	if theme, ok := update["theme_name"].(string); ok {
		cfg.ThemeName = theme
	}
	if useMock, ok := update["use_mock"].(bool); ok {
		cfg.UseMock = useMock
	}
	if htmlRich, ok := update["html_rich"].(bool); ok {
		cfg.HTMLRich = htmlRich
	}
	if imap, ok := update["imap"].(map[string]interface{}); ok {
		if host, ok := imap["host"].(string); ok {
			cfg.IMAP.Host = host
		}
		if port, ok := imap["port"].(float64); ok {
			cfg.IMAP.Port = int(port)
		}
		if user, ok := imap["username"].(string); ok {
			cfg.IMAP.Username = user
		}
		if pass, ok := imap["password"].(string); ok {
			cfg.IMAP.Password = pass
		}
		if ssl, ok := imap["ssl"].(bool); ok {
			cfg.IMAP.SSL = ssl
		}
	}
	if smtp, ok := update["smtp"].(map[string]interface{}); ok {
		if host, ok := smtp["host"].(string); ok {
			cfg.SMTP.Host = host
		}
		if port, ok := smtp["port"].(float64); ok {
			cfg.SMTP.Port = int(port)
		}
		if user, ok := smtp["username"].(string); ok {
			cfg.SMTP.Username = user
		}
		if pass, ok := smtp["password"].(string); ok {
			cfg.SMTP.Password = pass
		}
		if ssl, ok := smtp["ssl"].(bool); ok {
			cfg.SMTP.SSL = ssl
		}
	}

	if err := SaveConfig(cfg); err != nil {
		log.Printf("ERROR: SaveConfig failed: %v", err)
		http.Error(w, fmt.Sprintf("Failed to save file: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfg)
}

func handleGetFolders(w http.ResponseWriter, r *http.Request) {
	mailStoreMutex.Lock()
	defer mailStoreMutex.Unlock()

	unreadInbox := 0
	for _, e := range globalMailStore.Emails {
		if e.Folder == "inbox" && e.Unread {
			unreadInbox++
		}
	}

	response := map[string]interface{}{
		"folders": []map[string]interface{}{
			{"name": "inbox", "unread": unreadInbox},
			{"name": "starred"},
			{"name": "sent"},
			{"name": "drafts"},
			{"name": "archive"},
			{"name": "trash"},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleGetEmails(w http.ResponseWriter, r *http.Request) {
	folder := r.URL.Query().Get("folder")
	query := r.URL.Query().Get("q")

	cfg, _ := LoadConfig()

	mailStoreMutex.Lock()
	defer mailStoreMutex.Unlock()

	var emails []Email
	var err error

	if cfg.UseMock {
		if query != "" {
			emails = globalMailStore.SearchEmails(folder, query)
		} else {
			emails = globalMailStore.GetEmailsByFolder(folder)
		}
	} else {
		// Live Sync
		emails, err = FetchEmailsFromIMAP(cfg.IMAP, folder)
		if err != nil {
			http.Error(w, fmt.Sprintf("IMAP Fetch Error: %v", err), http.StatusInternalServerError)
			return
		}

		// Cache server emails in local memory representation
		for _, email := range emails {
			found := false
			for i, cached := range globalMailStore.Emails {
				if cached.ID == email.ID {
					globalMailStore.Emails[i] = email
					found = true
					break
				}
			}
			if !found {
				globalMailStore.Emails = append(globalMailStore.Emails, email)
			}
		}

		// Apply query filtering in server mode
		if query != "" {
			query = strings.ToLower(query)
			var filtered []Email
			for _, e := range emails {
				if strings.Contains(strings.ToLower(e.From), query) ||
					strings.Contains(strings.ToLower(e.Subject), query) ||
					strings.Contains(strings.ToLower(e.Body), query) {
					filtered = append(filtered, e)
				}
			}
			emails = filtered
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"emails": emails,
	})
}

func handleGetEmail(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	mailStoreMutex.Lock()
	defer mailStoreMutex.Unlock()

	var found *Email
	for i, e := range globalMailStore.Emails {
		if e.ID == id {
			found = &globalMailStore.Emails[i]
			break
		}
	}

	if found == nil {
		http.Error(w, "Email not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"email": found,
	})
}

type ActionPayload struct {
	Action string `json:"action"`
	ID     string `json:"id"`
	Value  bool   `json:"value"`
}

func handleAction(w http.ResponseWriter, r *http.Request) {
	var payload ActionPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	mailStoreMutex.Lock()
	defer mailStoreMutex.Unlock()

	if payload.Action == "star" {
		globalMailStore.ToggleStar(payload.ID)
	} else if payload.Action == "read" {
		globalMailStore.MarkAsRead(payload.ID, !payload.Value)
	} else if payload.Action == "delete" {
		globalMailStore.MoveToFolder(payload.ID, "trash")
	}

	w.WriteHeader(http.StatusOK)
}

type SendPayload struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

func handleSend(w http.ResponseWriter, r *http.Request) {
	var payload SendPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cfg, _ := LoadConfig()

	newMail := Email{
		ID:      fmt.Sprintf("composed-%d", time.Now().UnixNano()),
		From:    cfg.SMTP.Username,
		To:      payload.To,
		Subject: payload.Subject,
		Body:    payload.Body,
		Date:    time.Now(),
		Unread:  false,
		Starred: false,
		Folder:  "sent",
	}

	mailStoreMutex.Lock()
	defer mailStoreMutex.Unlock()

	if cfg.UseMock {
		globalMailStore.AddEmail(newMail)
	} else {
		err := SendEmailViaSMTP(cfg.SMTP, newMail)
		if err != nil {
			http.Error(w, fmt.Sprintf("SMTP Send Error: %v", err), http.StatusInternalServerError)
			return
		}
		globalMailStore.AddEmail(newMail)
	}

	w.WriteHeader(http.StatusOK)
}
