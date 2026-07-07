package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

// Email represents a single email message
type Email struct {
	ID      string    `json:"id"`
	From    string    `json:"from"`
	To      string    `json:"to"`
	Subject string    `json:"subject"`
	Body    string    `json:"body"`
	HTML    string    `json:"html"`
	IsHTML  bool      `json:"is_html"`
	Date    time.Time `json:"date"`
	Unread  bool      `json:"unread"`
	Starred bool      `json:"starred"`
	Folder  string    `json:"folder"` // "inbox", "starred", "sent", "drafts", "archive", "trash"
}

// MailStore manages the local collection of emails
type MailStore struct {
	Emails []Email
}

// NewMailStore creates and initializes a mail store with mock data
func NewMailStore() *MailStore {
	store := &MailStore{}
	store.LoadMockData()
	return store
}

// GetEmailsByFolder returns emails belonging to a specific folder
func (ms *MailStore) GetEmailsByFolder(folder string) []Email {
	var result []Email
	for _, email := range ms.Emails {
		if folder == "starred" {
			if email.Starred && email.Folder != "trash" {
				result = append(result, email)
			}
		} else if email.Folder == folder {
			result = append(result, email)
		}
	}
	// Sort by date descending (newest first)
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i].Date.Before(result[j].Date) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result
}

// MarkAsRead marks an email as read
func (ms *MailStore) MarkAsRead(id string, read bool) {
	for i, email := range ms.Emails {
		if email.ID == id {
			ms.Emails[i].Unread = !read
			break
		}
	}
}

// ToggleStar toggles the starred status of an email
func (ms *MailStore) ToggleStar(id string) {
	for i, email := range ms.Emails {
		if email.ID == id {
			ms.Emails[i].Starred = !ms.Emails[i].Starred
			break
		}
	}
}

// MoveToFolder moves an email to a different folder
func (ms *MailStore) MoveToFolder(id string, folder string) {
	for i, email := range ms.Emails {
		if email.ID == id {
			if email.Folder == "trash" && folder == "trash" {
				ms.Emails = append(ms.Emails[:i], ms.Emails[i+1:]...)
			} else {
				ms.Emails[i].Folder = folder
			}
			break
		}
	}
}

// SearchEmails searches emails matching a query in their sender, subject, or body
func (ms *MailStore) SearchEmails(folder string, query string) []Email {
	if query == "" {
		return ms.GetEmailsByFolder(folder)
	}

	query = strings.ToLower(query)
	var result []Email
	folderEmails := ms.GetEmailsByFolder(folder)

	for _, email := range folderEmails {
		if strings.Contains(strings.ToLower(email.From), query) ||
			strings.Contains(strings.ToLower(email.Subject), query) ||
			strings.Contains(strings.ToLower(email.Body), query) {
			result = append(result, email)
		}
	}
	return result
}

// AddEmail adds a new email to the store
func (ms *MailStore) AddEmail(email Email) {
	ms.Emails = append(ms.Emails, email)
}

// LoadMockData generates rich mock emails for previewing
func (ms *MailStore) LoadMockData() {
	now := time.Now()

	ms.Emails = []Email{
		{
			ID:      "1",
			From:    "sarah.cone@designco.io",
			To:      "you@example.com",
			Subject: "🎨 Initial Wireframes for KankerMail v2.0",
			Body: `Hi team,

I've put together the initial designs and wireframes for KankerMail v2.0!
We are moving towards a sleek pane-based design inspired by modern terminal interfaces.

Key highlights of the new UI:
- Sidebar with immediate visual cues for Unread and Starred counts.
- High-contrast, dynamic themes selectable on the fly (Neon, Dracula, Catppuccin, Nord).
- Interactive modals for email composition.
- Quick navigation keys matching Vim motions (j/k to scroll, Tab to change focus).

Please review the attached mocks and let me know your thoughts by Friday.

Best regards,
Sarah Cone
Lead UX Designer`,
			HTML: `<h1>🎨 Initial Wireframes for KankerMail v2.0</h1>
<p>Hi team,</p>
<p>I've put together the <strong>initial designs and wireframes</strong> for KankerMail v2.0! We are moving towards a sleek pane-based design inspired by modern terminal interfaces.</p>
<h3>Key Highlights of the new UI:</h3>
<ul>
  <li><strong>Sidebar</strong> with immediate visual cues for Unread and Starred counts.</li>
  <li><strong>High-contrast, dynamic themes</strong> selectable on the fly (Neon, Dracula, Catppuccin, Nord).</li>
  <li><strong>Interactive modals</strong> for email composition.</li>
  <li><strong>Quick navigation keys</strong> matching Vim motions (<code>j/k</code> to scroll, <code>Tab</code> to change focus).</li>
</ul>
<p>Please review the <a href="https://figma.com/file/mock-kankermail">attached Figma mocks</a> and let me know your thoughts by <strong>Friday</strong>.</p>
<p>Best regards,<br>
<strong>Sarah Cone</strong><br>
<em>Lead UX Designer</em></p>`,
			IsHTML:  true,
			Date:    now.Add(-10 * time.Minute),
			Unread:  true,
			Starred: true,
			Folder:  "inbox",
		},
		{
			ID:      "2",
			From:    "alerts@kubernetes-cluster.internal",
			To:      "infra-team@example.com",
			Subject: "⚠️ Warning: CPU usage high on node-04b",
			Body: `[METRICS REPORT - CRITICAL REGION]
Host: node-04b.prod.backend
Service: api-gateway
Metric: CPU Utilization
Current Value: 91.4%
Threshold: 85.0%
Duration: 5 minutes 12 seconds

Action taken:
- Automated scaling triggered: spinning up node-05a.
- Traffic re-routing initiated to load balance traffic.
- Memory profile captured for analysis.

Please check the Grafana dashboard at https://grafana.internal/prod-cluster for real-time visualization and to investigate potential memory leaks in the Go routing stack.`,
			HTML: `<h2>⚠️ Warning: CPU usage high on node-04b</h2>
<hr>
<p><strong>[METRICS REPORT - CRITICAL REGION]</strong></p>
<table border="1" cellpadding="5">
  <tr><td><strong>Host</strong></td><td>node-04b.prod.backend</td></tr>
  <tr><td><strong>Service</strong></td><td>api-gateway</td></tr>
  <tr><td><strong>Metric</strong></td><td>CPU Utilization</td></tr>
  <tr><td><strong>Current Value</strong></td><td><font color="red"><b>91.4%</b></font></td></tr>
  <tr><td><strong>Threshold</strong></td><td>85.0%</td></tr>
  <tr><td><strong>Duration</strong></td><td>5 minutes 12 seconds</td></tr>
</table>
<h3>Action taken:</h3>
<ul>
  <li>Automated scaling triggered: spinning up node-05a.</li>
  <li>Traffic re-routing initiated to load balance traffic.</li>
  <li>Memory profile captured for analysis.</li>
</ul>
<p>Please check the <a href="https://grafana.internal/prod-cluster">Grafana dashboard</a> for real-time visualization and to investigate potential memory leaks in the Go routing stack.</p>`,
			IsHTML:  true,
			Date:    now.Add(-45 * time.Minute),
			Unread:  true,
			Starred: false,
			Folder:  "inbox",
		},
		{
			ID:      "3",
			From:    "newsletter@tldr-tech.com",
			To:      "subscribers@tldr-tech.com",
			Subject: "🚀 TLDR Tech: Go 1.27 Release Candidates & The Rise of TUIs",
			Body: `TLDR Tech - July 7, 2026

Go 1.27 Release Candidate Released (5 min read)
----------------------------------------------
The Go team has released the first candidate for Go 1.27. It brings significant improvements to compiling speeds, further reduction in GC latency, and expanded compiler support for loop optimizations. The standard library gains new structured logging capabilities (slog v2) and extended crypto protocols.

Why TUIs are making a comeback in 2026 (3 min read)
--------------------------------------------------
Developers are increasingly returning to Terminal User Interfaces (TUIs). Standard CLI apps are being replaced by responsive, styled widgets built using libraries like Charm's Bubble Tea in Go or Textual in Python. The appeal lies in mouse-free efficiency, lightning-fast load times, and their visual aesthetic that fits perfectly in modern developers' workspaces.

AI Hardware Startup raises $400M (2 min read)
---------------------------------------------
Another AI silicon startup has raised a massive Series B to manufacture analog processors aiming to drop LLM inference costs by an order of magnitude. Tests show impressive energy efficiency but programmability remains the main blocker for software adoption.`,
			HTML: `<h1>🚀 TLDR Tech</h1>
<p><em>July 7, 2026</em></p>
<hr>
<h3>Go 1.27 Release Candidate Released (5 min read)</h3>
<p>The Go team has released the first candidate for Go 1.27. It brings significant improvements to compiling speeds, further reduction in GC latency, and expanded compiler support for loop optimizations. The standard library gains new structured logging capabilities (<code>slog v2</code>) and extended crypto protocols.</p>
<h3>Why TUIs are making a comeback in 2026 (3 min read)</h3>
<p>Developers are increasingly returning to <strong>Terminal User Interfaces (TUIs)</strong>. Standard CLI apps are being replaced by responsive, styled widgets built using libraries like Charm's Bubble Tea in Go or Textual in Python. The appeal lies in mouse-free efficiency, lightning-fast load times, and their visual aesthetic that fits perfectly in modern developers' workspaces.</p>
<h3>AI Hardware Startup raises $400M (2 min read)</h3>
<p>Another AI silicon startup has raised a massive Series B to manufacture analog processors aiming to drop LLM inference costs by an order of magnitude. Tests show impressive energy efficiency but programmability remains the main blocker for software adoption.</p>`,
			IsHTML:  true,
			Date:    now.Add(-2 * time.Hour),
			Unread:  false,
			Starred: true,
			Folder:  "inbox",
		},
		{
			ID:      "4",
			From:    "alex.roadmap@productlabs.com",
			To:      "you@example.com",
			Subject: "📅 Sync on Q3 planning & timeline adjustment",
			Body: `Hey there,

Can we schedule a quick 15-minute call tomorrow morning to talk about the Q3 feature list? 
We need to lock down our deliverables for the next sprint.

Currently proposed items:
1. Integration of real IMAP/SMTP configurations.
2. Search and filter optimization for large Maildirs.
3. Multi-account support.

Let me know what time works best for you.

Thanks,
Alex`,
			HTML:    "",
			IsHTML:  false,
			Date:    now.Add(-5 * time.Hour),
			Unread:  false,
			Starred: false,
			Folder:  "inbox",
		},
		{
			ID:      "5",
			From:    "github-notifications@github.com",
			To:      "you@example.com",
			Subject: "[GitHub] PR #412 Merged: Implement fuzzy search on email subjects",
			Body: `pr-bot merged commit 8fa12d9 into master.

Reviewers: @gopher-dev (Approved), @tui-tester (Approved)

Description of changes:
- Added a search bar in the list view that filters emails using substring matching.
- Performance optimization: index fields to keep search fast in directories with 10k+ items.
- Fixed boundary crash when search result was empty.

You can view the full discussion and review log at:
https://github.com/example/kankermail/pull/412`,
			HTML: `<h3>[GitHub] PR #412 Merged: Implement fuzzy search on email subjects</h3>
<p><code>pr-bot</code> merged commit <code>8fa12d9</code> into master.</p>
<p><b>Reviewers:</b> @gopher-dev (Approved), @tui-tester (Approved)</p>
<p><b>Description of changes:</b></p>
<ul>
  <li>Added a search bar in the list view that filters emails using substring matching.</li>
  <li>Performance optimization: index fields to keep search fast in directories with 10k+ items.</li>
  <li>Fixed boundary crash when search result was empty.</li>
</ul>
<p>You can view the full discussion and review log at:<br>
<a href="https://github.com/example/kankermail/pull/412">https://github.com/example/kankermail/pull/412</a></p>`,
			IsHTML:  true,
			Date:    now.Add(-24 * time.Hour),
			Unread:  false,
			Starred: false,
			Folder:  "archive",
		},
		{
			ID:      "6",
			From:    "you@example.com",
			To:      "sarah.cone@designco.io",
			Subject: "Re: 🎨 Initial Wireframes for KankerMail v2.0",
			Body: `Hi Sarah,

These designs look fantastic! I love the neon dark preset and the pane layout.

One request: can we make sure that shifting focus between panes feels natural? Tab and Shift+Tab are perfect, but maybe arrow keys could work too when we are at the borders.

I will start prototyping the Lip Gloss style structures today.

Best,
Me`,
			HTML:    "",
			IsHTML:  false,
			Date:    now.Add(-5 * time.Minute),
			Unread:  false,
			Starred: false,
			Folder:  "sent",
		},
	}
}

// ============================================================================
// REAL IMAP / SMTP IMPLEMENTATION
// ============================================================================

// mapFolderToIMAP translates local folder names into standard IMAP folder names
func mapFolderToIMAP(folder string) string {
	switch folder {
	case "inbox":
		return "INBOX"
	case "sent":
		return "Sent Items"
	case "trash":
		return "Deleted Items"
	case "archive":
		return "Archive"
	case "drafts":
		return "Drafts"
	default:
		return "INBOX"
	}
}

// FetchEmailsFromIMAP connects to the IMAP server and fetches recent emails
func FetchEmailsFromIMAP(cfg IMAPConfig, folder string) ([]Email, error) {
	if cfg.Username == "" || cfg.Password == "" {
		return nil, fmt.Errorf("credentials not provided in config file")
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	
	var c *client.Client
	var err error

	if cfg.SSL {
		c, err = client.DialTLS(addr, nil)
	} else {
		c, err = client.Dial(addr)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %v", err)
	}
	defer c.Logout()

	// Login
	if err := c.Login(cfg.Username, cfg.Password); err != nil {
		return nil, fmt.Errorf("authentication failed: %v", err)
	}

	// Select Mailbox Folder
	imapFolder := mapFolderToIMAP(folder)
	mbox, err := c.Select(imapFolder, true) // Read-only
	if err != nil {
		imapFolder = "INBOX"
		mbox, err = c.Select(imapFolder, true)
		if err != nil {
			return nil, fmt.Errorf("failed to select folder %s: %v", folder, err)
		}
	}

	if mbox.Messages == 0 {
		return []Email{}, nil
	}

	// Fetch the last 20 emails
	var from uint32 = 1
	if mbox.Messages > 20 {
		from = mbox.Messages - 19
	}
	to := mbox.Messages

	seqset := new(imap.SeqSet)
	seqset.AddRange(from, to)

	// We want to fetch the envelope, flags, and the email body sections
	var section imap.BodySectionName
	items := []imap.FetchItem{section.FetchItem(), imap.FetchEnvelope, imap.FetchFlags}

	messages := make(chan *imap.Message, 20)
	done := make(chan error, 1)

	go func() {
		done <- c.Fetch(seqset, items, messages)
	}()

	var fetchedEmails []Email

	for msg := range messages {
		unread := true
		for _, flag := range msg.Flags {
			if flag == imap.SeenFlag {
				unread = false
				break
			}
		}

		starred := false
		for _, flag := range msg.Flags {
			if flag == imap.FlaggedFlag {
				starred = true
				break
			}
		}

		subject := "No Subject"
		fromStr := "Unknown Sender"
		toStr := "Unknown Recipient"
		date := time.Now()

		if msg.Envelope != nil {
			subject = msg.Envelope.Subject
			date = msg.Envelope.Date
			if len(msg.Envelope.From) > 0 {
				f := msg.Envelope.From[0]
				if f.PersonalName != "" {
					fromStr = fmt.Sprintf("%s <%s@%s>", f.PersonalName, f.MailboxName, f.HostName)
				} else {
					fromStr = fmt.Sprintf("%s@%s", f.MailboxName, f.HostName)
				}
			}
			if len(msg.Envelope.To) > 0 {
				t := msg.Envelope.To[0]
				if t.PersonalName != "" {
					toStr = fmt.Sprintf("%s <%s@%s>", t.PersonalName, t.MailboxName, t.HostName)
				} else {
					toStr = fmt.Sprintf("%s@%s", t.MailboxName, t.HostName)
				}
			}
		}

		// Read and parse body content
		body := ""
		htmlContent := ""
		isHTML := false
		bodyReader := msg.GetBody(&section)
		if bodyReader != nil {
			if parsedMsg, err := mail.ReadMessage(bodyReader); err == nil {
				body, htmlContent, isHTML = parseEmailBodyAndHTML(parsedMsg)
			}
		}

		fetchedEmails = append(fetchedEmails, Email{
			ID:      fmt.Sprintf("%d", msg.SeqNum),
			From:    fromStr,
			To:      toStr,
			Subject: subject,
			Body:    body,
			HTML:    htmlContent,
			IsHTML:  isHTML,
			Date:    date,
			Unread:  unread,
			Starred: starred,
			Folder:  folder,
		})
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("fetch process failed: %v", err)
	}

	// Reverse to have newest first
	for i, j := 0, len(fetchedEmails)-1; i < j; i, j = i+1, j-1 {
		fetchedEmails[i], fetchedEmails[j] = fetchedEmails[j], fetchedEmails[i]
	}

	return fetchedEmails, nil
}

// parseEmailBodyAndHTML extracts plain text and HTML bodies from a message
func parseEmailBodyAndHTML(m *mail.Message) (string, string, bool) {
	contentType := m.Header.Get("Content-Type")
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		b, _ := io.ReadAll(m.Body)
		return string(b), "", false
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		boundary, ok := params["boundary"]
		if !ok {
			b, _ := io.ReadAll(m.Body)
			return string(b), "", false
		}

		mr := multipart.NewReader(m.Body, boundary)
		var plainParts []string
		var htmlParts []string

		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				break
			}

			partContentType := part.Header.Get("Content-Type")
			partMediaType, partParams, _ := mime.ParseMediaType(partContentType)

			if strings.HasPrefix(partMediaType, "multipart/") {
				nestedBoundary, ok := partParams["boundary"]
				if ok {
					nestedReader := multipart.NewReader(part, nestedBoundary)
					for {
						np, err := nestedReader.NextPart()
						if err == io.EOF {
							break
						}
						if err != nil {
							break
						}
						npType, _, _ := mime.ParseMediaType(np.Header.Get("Content-Type"))
						if npType == "text/plain" {
							b, _ := io.ReadAll(np)
							plainParts = append(plainParts, string(b))
						} else if npType == "text/html" {
							b, _ := io.ReadAll(np)
							htmlParts = append(htmlParts, string(b))
						}
					}
				}
			} else if partMediaType == "text/plain" {
				b, _ := io.ReadAll(part)
				plainParts = append(plainParts, string(b))
			} else if partMediaType == "text/html" {
				b, _ := io.ReadAll(part)
				htmlParts = append(htmlParts, string(b))
			}
		}

		plainBody := strings.Join(plainParts, "\n\n")
		htmlBody := strings.Join(htmlParts, "\n\n")

		return plainBody, htmlBody, len(htmlParts) > 0
	}

	b, _ := io.ReadAll(m.Body)
	if mediaType == "text/html" {
		return "", string(b), true
	}
	return string(b), "", false
}

// SendEmailViaSMTP sends an email through SMTP with SSL or STARTTLS support
func SendEmailViaSMTP(cfg SMTPConfig, email Email) error {
	if cfg.Username == "" || cfg.Password == "" {
		return fmt.Errorf("credentials not provided in config file")
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)

	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("From: %s\r\n", cfg.Username))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", email.To))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", email.Subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(email.Body)

	if cfg.Port == 465 {
		tlsconfig := &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         cfg.Host,
		}
		conn, err := tls.Dial("tcp", addr, tlsconfig)
		if err != nil {
			return fmt.Errorf("TLS dial to SMTP host failed: %v", err)
		}
		defer conn.Close()

		c, err := smtp.NewClient(conn, cfg.Host)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %v", err)
		}
		defer c.Close()

		if err = c.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth failed: %v", err)
		}
		if err = c.Mail(cfg.Username); err != nil {
			return err
		}
		if err = c.Rcpt(email.To); err != nil {
			return err
		}
		w, err := c.Data()
		if err != nil {
			return err
		}
		_, err = w.Write([]byte(msg.String()))
		if err != nil {
			return err
		}
		err = w.Close()
		if err != nil {
			return err
		}
		return c.Quit()
	}

	err := smtp.SendMail(addr, auth, cfg.Username, []string{email.To}, []byte(msg.String()))
	if err != nil {
		return fmt.Errorf("SMTP SendMail failed: %v", err)
	}
	return nil
}
