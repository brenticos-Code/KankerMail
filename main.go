package main

import (
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/jaytaylor/html2text"
	"github.com/muesli/reflow/truncate"
	"github.com/muesli/reflow/wrap"
)

type activePane int

const (
	sidebarPane activePane = iota
	listPane
	viewPane
)

// Async messages for Bubble Tea
type imapFetchSuccessMsg struct {
	emails []Email
}

type imapFetchErrorMsg struct {
	err error
}

type smtpSendSuccessMsg struct{}

type smtpSendErrorMsg struct {
	err error
}

type model struct {
	config             Config
	mailStore          *MailStore
	emails             []Email // Current folder's emails (fetched or mock)
	activePane         activePane
	selectedFolderIdx  int
	selectedEmailIdx   int
	listOffset         int
	selectedSettingsIdx int // Settings list selection index
	searchQuery        string
	searching          bool
	searchInput        textinput.Model
	composer           Composer
	showComposer       bool
	loading            bool
	sending            bool
	viewport           viewport.Model
	statusMessage      string
	statusTime         time.Time
	width              int
	height             int
	mainWidth          int
	contentHeight      int
	listHeight         int
	viewHeight         int
	ready              bool
}

var folders = []string{"inbox", "starred", "sent", "drafts", "archive", "trash", "settings"}
var settingsItems = []string{
	"Toggle Mock Mode",
	"Toggle HTML Rendering",
	"Theme: Neon Dark",
	"Theme: Dracula",
	"Theme: Catppuccin Macchiato",
	"Theme: Nord",
}

// Helper commands for async operations
func fetchEmailsCmd(cfg IMAPConfig, folder string) tea.Cmd {
	return func() tea.Msg {
		emails, err := FetchEmailsFromIMAP(cfg, folder)
		if err != nil {
			return imapFetchErrorMsg{err: err}
		}
		return imapFetchSuccessMsg{emails: emails}
	}
}

func sendEmailCmd(cfg SMTPConfig, email Email) tea.Cmd {
	return func() tea.Msg {
		err := SendEmailViaSMTP(cfg, email)
		if err != nil {
			return smtpSendErrorMsg{err: err}
		}
		return smtpSendSuccessMsg{}
	}
}

func initialModel() model {
	cfg, _ := LoadConfig()
	SetTheme(cfg.ThemeName)
	ms := NewMailStore()

	si := textinput.New()
	si.Placeholder = "Search..."
	si.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(CurrentTheme.Accent))
	si.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(CurrentTheme.Accent))

	comp := NewComposer()

	// Initial emails list setup
	var initialEmails []Email
	if cfg.UseMock {
		initialEmails = ms.GetEmailsByFolder("inbox")
	}

	// Match settings theme selection index
	themeIdx := 2 // Default to 2 (Neon Dark) since item 0 is Mock, 1 is HTML Rendering
	for i, t := range AllThemes {
		if t.Name == cfg.ThemeName {
			themeIdx = i + 2 // Offset by 2 because index 0 is Mock, 1 is HTML Rendering
			break
		}
	}

	m := model{
		config:             cfg,
		mailStore:          ms,
		emails:             initialEmails,
		activePane:         sidebarPane,
		selectedFolderIdx:  0,
		selectedEmailIdx:   0,
		listOffset:         0,
		selectedSettingsIdx: themeIdx,
		searchInput:        si,
		composer:           comp,
		showComposer:       false,
		viewport:           viewport.New(0, 0),
		statusMessage:      "Welcome to KankerMail! Press [s] to search • [n] to compose.",
		statusTime:         time.Now(),
	}

	return m
}

func (m model) Init() tea.Cmd {
	if !m.config.UseMock {
		return tea.Batch(
			textinput.Blink,
			fetchEmailsCmd(m.config.IMAP, "inbox"),
		)
	}
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	res, cmd := m.updateInternal(msg)
	if nm, ok := res.(model); ok {
		nm.clampListOffset()
		return nm, cmd
	}
	return res, cmd
}

func (m *model) clampListOffset() {
	pageSize := m.listHeight - 3
	if pageSize < 1 {
		pageSize = 1
	}
	if m.selectedEmailIdx < m.listOffset {
		m.listOffset = m.selectedEmailIdx
	} else if m.selectedEmailIdx >= m.listOffset+pageSize {
		m.listOffset = m.selectedEmailIdx - pageSize + 1
	}
	emails := m.getActiveEmails()
	maxOffset := len(emails) - pageSize
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.listOffset > maxOffset {
		m.listOffset = maxOffset
	}
	if m.listOffset < 0 {
		m.listOffset = 0
	}
}

func (m model) updateInternal(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Math for sub-panes
		sidebarWidth := 22
		appWidth := m.width
		if appWidth > 130 {
			appWidth = 130
		}
		m.mainWidth = appWidth - sidebarWidth - 1
		if m.mainWidth < 20 {
			m.mainWidth = 20
		}
		m.contentHeight = m.height - 2 // space for status bar and trailing newline
		if m.contentHeight < 10 {
			m.contentHeight = 10
		}

		m.listHeight = (m.contentHeight * 4) / 10
		m.viewHeight = m.contentHeight - m.listHeight

		// Setup viewport for email body (accounting for borders)
		if !m.ready {
			m.viewport = viewport.New(m.mainWidth-2, m.viewHeight-2)
			m.ready = true
		} else {
			m.viewport.Width = m.mainWidth - 2
			m.viewport.Height = m.viewHeight - 2
		}

		m.composer.SetSize(m.width, m.height)
		m.updateEmailBody()
		return m, nil

	case imapFetchSuccessMsg:
		m.loading = false
		m.emails = msg.emails
		m.selectedEmailIdx = 0
		m.statusMessage = fmt.Sprintf("Successfully fetched %d emails from IMAP.", len(m.emails))
		m.updateEmailBody()
		return m, nil

	case imapFetchErrorMsg:
		m.loading = false
		m.emails = nil
		m.statusMessage = fmt.Sprintf("IMAP Connection Error: %v", msg.err)
		m.updateEmailBody()
		return m, nil

	case smtpSendSuccessMsg:
		m.sending = false
		m.showComposer = false
		m.composer.Reset()
		m.statusMessage = "Email sent successfully via SMTP!"
		// If in sent folder, sync it if not mock
		if folders[m.selectedFolderIdx] == "sent" && !m.config.UseMock {
			m.loading = true
			return m, fetchEmailsCmd(m.config.IMAP, "sent")
		}
		return m, nil

	case smtpSendErrorMsg:
		m.sending = false
		m.statusMessage = fmt.Sprintf("SMTP Error: %v", msg.err)
		return m, nil

	case tea.KeyMsg:
		keyStr := msg.String()

		// 1. COMPOSER MODE ACTIVE
		if m.showComposer {
			switch keyStr {
			case "esc":
				if !m.sending {
					m.showComposer = false
					m.composer.Reset()
					m.statusMessage = "Composition cancelled."
				}
				return m, nil
			case "ctrl+s":
				if m.sending {
					return m, nil
				}
				newMail := m.composer.GetEmail()
				newMail.ID = fmt.Sprintf("composed-%d", len(m.mailStore.Emails)+1)
				newMail.Date = time.Now()
				newMail.Unread = false
				newMail.Starred = false
				newMail.Folder = "sent"

				if m.config.UseMock {
					// Save locally in mock store
					m.mailStore.AddEmail(newMail)
					m.showComposer = false
					m.composer.Reset()
					m.statusMessage = "Email sent successfully! (Mock mode: saved locally)"
					if folders[m.selectedFolderIdx] == "sent" {
						m.emails = m.mailStore.GetEmailsByFolder("sent")
						m.selectedEmailIdx = 0
						m.updateEmailBody()
					}
					return m, nil
				} else {
					// Real SMTP send
					m.sending = true
					m.statusMessage = "Sending email via SMTP..."
					return m, sendEmailCmd(m.config.SMTP, newMail)
				}
			default:
				if !m.sending {
					m.composer, cmd = m.composer.Update(msg)
					return m, cmd
				}
				return m, nil
			}
		}

		// 2. SEARCH MODE ACTIVE
		if m.searching {
			switch keyStr {
			case "enter":
				m.searching = false
				m.searchQuery = m.searchInput.Value()
				m.selectedEmailIdx = 0
				m.updateEmailBody()
				m.activePane = listPane
				m.statusMessage = fmt.Sprintf("Search locked. Results for: '%s'", m.searchQuery)
				return m, nil
			case "esc":
				m.searching = false
				m.searchInput.SetValue("")
				m.searchQuery = ""
				m.selectedEmailIdx = 0
				m.updateEmailBody()
				m.statusMessage = "Search cleared."
				return m, nil
			default:
				m.searchInput, cmd = m.searchInput.Update(msg)
				m.searchQuery = m.searchInput.Value()
				m.selectedEmailIdx = 0
				m.updateEmailBody()
				return m, cmd
			}
		}

		// 3. NORMAL TUI KEYBINDINGS
		switch keyStr {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "ctrl+r":
			// Refresh / Sync active folder from server
			folderName := folders[m.selectedFolderIdx]
			if folderName != "settings" {
				if m.config.UseMock {
					m.emails = m.mailStore.GetEmailsByFolder(folderName)
					m.selectedEmailIdx = 0
					m.updateEmailBody()
					m.statusMessage = "Refreshed mock emails."
				} else {
					m.loading = true
					m.statusMessage = fmt.Sprintf("Refreshing %s from IMAP...", folderName)
					return m, fetchEmailsCmd(m.config.IMAP, folderName)
				}
			}
			return m, nil

		case "tab":
			if folders[m.selectedFolderIdx] == "settings" {
				if m.activePane == sidebarPane {
					m.activePane = listPane
				} else {
					m.activePane = sidebarPane
				}
			} else {
				m.activePane = (m.activePane + 1) % 3
			}
			return m, nil

		case "shift+tab":
			if folders[m.selectedFolderIdx] == "settings" {
				if m.activePane == sidebarPane {
					m.activePane = listPane
				} else {
					m.activePane = sidebarPane
				}
			} else {
				m.activePane = (m.activePane - 1 + 3) % 3
			}
			return m, nil

		case "j", "down":
			if m.activePane == viewPane {
				m.viewport, cmd = m.viewport.Update(msg)
				return m, cmd
			}
			m.handleDown()
			// If not in sidebar mode, or in mock mode, load content instantly
			if m.activePane != sidebarPane || m.config.UseMock {
				m.updateEmailBody()
			}
			return m, nil

		case "k", "up":
			if m.activePane == viewPane {
				m.viewport, cmd = m.viewport.Update(msg)
				return m, cmd
			}
			m.handleUp()
			if m.activePane != sidebarPane || m.config.UseMock {
				m.updateEmailBody()
			}
			return m, nil

		case "enter":
			if m.activePane == sidebarPane {
				m.selectedEmailIdx = 0
				m.searchQuery = ""
				m.searchInput.SetValue("")
				folderName := folders[m.selectedFolderIdx]

				if folderName == "settings" {
					m.activePane = listPane
					return m, nil
				}

				m.activePane = listPane
				if !m.config.UseMock {
					m.loading = true
					m.statusMessage = fmt.Sprintf("Fetching %s from IMAP...", folderName)
					m.emails = nil
					m.updateEmailBody()
					return m, fetchEmailsCmd(m.config.IMAP, folderName)
				} else {
					m.emails = m.mailStore.GetEmailsByFolder(folderName)
					m.updateEmailBody()
				}
			} else if folders[m.selectedFolderIdx] == "settings" && m.activePane == listPane {
				// Handle settings items selections
				if m.selectedSettingsIdx == 0 {
					// Toggle mock mode
					m.config.UseMock = !m.config.UseMock
					_ = SaveConfig(m.config)

					if m.config.UseMock {
						m.emails = m.mailStore.GetEmailsByFolder("inbox")
						m.selectedEmailIdx = 0
						m.selectedFolderIdx = 0
						m.updateEmailBody()
						m.statusMessage = "Enabled Mock Mode. Loaded local simulator."
					} else {
						m.emails = nil
						m.selectedEmailIdx = 0
						m.selectedFolderIdx = 0
						m.loading = true
						m.statusMessage = "Mock Mode disabled. Fetching INBOX from server..."
						return m, fetchEmailsCmd(m.config.IMAP, "inbox")
					}
				} else if m.selectedSettingsIdx == 1 {
					// Toggle HTML rendering
					m.config.HTMLRich = !m.config.HTMLRich
					_ = SaveConfig(m.config)
					m.updateEmailBody()
					m.statusMessage = fmt.Sprintf("HTML Rich Rendering set to %t", m.config.HTMLRich)
				} else {
					// Theme selection index (offset by 2)
					themeIdx := m.selectedSettingsIdx - 2
					selectedTheme := AllThemes[themeIdx]
					SetTheme(selectedTheme.Name)
					m.config.ThemeName = selectedTheme.Name
					_ = SaveConfig(m.config)
					m.updateEmailBody() // Refresh viewer styled markup
					m.statusMessage = fmt.Sprintf("Applied theme: %s", selectedTheme.Name)
				}
			} else if m.activePane == listPane {
				m.activePane = viewPane
			}
			return m, nil

		case "n":
			m.showComposer = true
			m.composer.Reset()
			m.statusMessage = "Composing email..."
			return m, nil

		case "r":
			emails := m.getActiveEmails()
			if len(emails) > 0 && m.selectedEmailIdx < len(emails) {
				email := emails[m.selectedEmailIdx]
				m.showComposer = true
				m.composer.Reset()
				m.composer.toInput.SetValue(email.From)
				m.composer.subjectInput.SetValue("Re: " + strings.TrimPrefix(email.Subject, "Re: "))
				m.composer.bodyInput.SetValue(fmt.Sprintf("\n\nOn %s, %s wrote:\n> %s",
					email.Date.Format("Mon Jan _2, 2006 at 3:04PM"),
					email.From,
					strings.ReplaceAll(email.Body, "\n", "\n> ")))
				m.composer.focusIndex = 2 // focus body input directly
				m.composer.updateFocus()
				m.statusMessage = "Replying to email..."
			}
			return m, nil

		case "s":
			m.searching = true
			m.searchInput.Focus()
			m.statusMessage = "Type search term and press Enter..."
			return m, nil

		case "d":
			emails := m.getActiveEmails()
			if len(emails) > 0 && m.selectedEmailIdx < len(emails) {
				email := emails[m.selectedEmailIdx]
				folder := folders[m.selectedFolderIdx]

				if m.config.UseMock {
					if folder == "trash" {
						m.mailStore.MoveToFolder(email.ID, "trash")
						m.statusMessage = "Permanently deleted email."
					} else {
						m.mailStore.MoveToFolder(email.ID, "trash")
						m.statusMessage = "Moved email to Trash."
					}
					m.emails = m.mailStore.GetEmailsByFolder(folder)
					if m.selectedEmailIdx >= len(m.emails) && m.selectedEmailIdx > 0 {
						m.selectedEmailIdx--
					}
					m.updateEmailBody()
				} else {
					m.statusMessage = "Delete operation is simulated locally in server mode."
					// Remove from active list in memory
					for i, e := range m.emails {
						if e.ID == email.ID {
							m.emails = append(m.emails[:i], m.emails[i+1:]...)
							break
						}
					}
					if m.selectedEmailIdx >= len(m.getActiveEmails()) && m.selectedEmailIdx > 0 {
						m.selectedEmailIdx--
					}
					m.updateEmailBody()
				}
			}
			return m, nil

		case "f", "*":
			emails := m.getActiveEmails()
			if len(emails) > 0 && m.selectedEmailIdx < len(emails) {
				email := emails[m.selectedEmailIdx]
				if m.config.UseMock {
					m.mailStore.ToggleStar(email.ID)
					m.statusMessage = "Toggled star status."
				} else {
					// Toggle local memory representation
					for i, e := range m.emails {
						if e.ID == email.ID {
							m.emails[i].Starred = !m.emails[i].Starred
							break
						}
					}
					m.statusMessage = "Starred status toggled locally."
				}
			}
			return m, nil

		case "u":
			emails := m.getActiveEmails()
			if len(emails) > 0 && m.selectedEmailIdx < len(emails) {
				email := emails[m.selectedEmailIdx]
				if m.config.UseMock {
					m.mailStore.MarkAsRead(email.ID, !email.Unread)
					m.statusMessage = "Toggled unread status."
				} else {
					// Toggle local memory representation
					for i, e := range m.emails {
						if e.ID == email.ID {
							m.emails[i].Unread = !m.emails[i].Unread
							break
						}
					}
					m.statusMessage = "Unread status toggled locally."
				}
			}
			return m, nil
		}
	}

	// Scroll viewport in viewing pane if it's active
	if m.activePane == viewPane && !m.showComposer && !m.searching {
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *model) handleUp() {
	switch m.activePane {
	case sidebarPane:
		if m.selectedFolderIdx > 0 {
			m.selectedFolderIdx--
			m.selectedEmailIdx = 0
			if m.config.UseMock {
				m.emails = m.mailStore.GetEmailsByFolder(folders[m.selectedFolderIdx])
			}
		}
	case listPane:
		if folders[m.selectedFolderIdx] == "settings" {
			if m.selectedSettingsIdx > 0 {
				m.selectedSettingsIdx--
			}
		} else {
			if m.selectedEmailIdx > 0 {
				m.selectedEmailIdx--
			}
		}
	}
}

func (m *model) handleDown() {
	switch m.activePane {
	case sidebarPane:
		if m.selectedFolderIdx < len(folders)-1 {
			m.selectedFolderIdx++
			m.selectedEmailIdx = 0
			if m.config.UseMock {
				m.emails = m.mailStore.GetEmailsByFolder(folders[m.selectedFolderIdx])
			}
		}
	case listPane:
		if folders[m.selectedFolderIdx] == "settings" {
			if m.selectedSettingsIdx < len(settingsItems)-1 {
				m.selectedSettingsIdx++
			}
		} else {
			emails := m.getActiveEmails()
			if m.selectedEmailIdx < len(emails)-1 {
				m.selectedEmailIdx++
			}
		}
	}
}

func (m *model) getActiveEmails() []Email {
	if m.config.UseMock {
		folder := folders[m.selectedFolderIdx]
		return m.mailStore.SearchEmails(folder, m.searchQuery)
	}

	// Real IMAP filter locally
	if m.searchQuery == "" {
		return m.emails
	}

	query := strings.ToLower(m.searchQuery)
	var result []Email
	for _, email := range m.emails {
		if strings.Contains(strings.ToLower(email.From), query) ||
			strings.Contains(strings.ToLower(email.Subject), query) ||
			strings.Contains(strings.ToLower(email.Body), query) {
			result = append(result, email)
		}
	}
	// Wait, let me make sure I write append(result, email) below:
	return result
}

func (m *model) updateEmailBody() {
	emails := m.getActiveEmails()
	if len(emails) == 0 || m.selectedEmailIdx >= len(emails) {
		m.viewport.SetContent("No messages in this folder matching filters.")
		return
	}

	email := emails[m.selectedEmailIdx]

	// Mark as read locally
	if email.Unread {
		if m.config.UseMock {
			m.mailStore.MarkAsRead(email.ID, true)
		} else {
			for i, e := range m.emails {
				if e.ID == email.ID {
					m.emails[i].Unread = false
					break
				}
			}
		}
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("From:    %s\n", email.From))
	builder.WriteString(fmt.Sprintf("To:      %s\n", email.To))
	builder.WriteString(fmt.Sprintf("Date:    %s\n", email.Date.Format("Mon Jan _2, 2006 at 3:04PM")))
	builder.WriteString(fmt.Sprintf("Subject: %s\n", email.Subject))
	
	if m.viewport.Width > 0 {
		builder.WriteString(strings.Repeat("─", m.viewport.Width) + "\n")
	} else {
		builder.WriteString("─\n")
	}

	bodyToShow := email.Body
	if m.config.HTMLRich && email.IsHTML && email.HTML != "" {
		// Convert HTML to text with basic markdown styling
		markdownText, err := html2text.FromString(email.HTML, html2text.Options{TextOnly: false})
		if err == nil {
			glStyle := GetGlamourStyle()
			widthLimit := m.viewport.Width - 4
			if widthLimit < 20 {
				widthLimit = 20
			}
			r, err := glamour.NewTermRenderer(
				glamour.WithStandardStyle(glStyle),
				glamour.WithWordWrap(widthLimit),
			)
			if err == nil {
				formattedBody, err := r.Render(markdownText)
				if err == nil {
					bodyToShow = formattedBody
				}
			}
		}
	}

	builder.WriteString(bodyToShow)
	
	wrapWidth := m.viewport.Width
	if wrapWidth < 10 {
		wrapWidth = 10
	}
	wrappedContent := wrap.String(builder.String(), wrapWidth)
	
	var finalLines []string
	for _, line := range strings.Split(wrappedContent, "\n") {
		finalLines = append(finalLines, truncate.String(line, uint(wrapWidth)))
	}
	m.viewport.SetContent(strings.Join(finalLines, "\n"))
}

// VIEW LOGIC
func (m model) View() string {
	if !m.ready {
		return "Initializing KankerMail..."
	}

	styles := GetStyles()

	// 1. RENDER OVERLAYS
	if m.showComposer {
		return m.composer.View(styles)
	}

	// 2. LAYOUT PREPARATION
	sidebarWidth := 22
	appWidth := m.mainWidth + sidebarWidth + 1

	// Sidebar View
	sidebarStr := m.renderSidebar(styles, sidebarWidth, m.contentHeight)

	// Main View Pane
	var mainStr string
	folder := folders[m.selectedFolderIdx]

	if folder == "settings" {
		mainStr = m.renderSettings(styles, m.mainWidth, m.contentHeight)
	} else {
		// Normal folder mail list & view
		listStr := m.renderEmailList(styles, m.mainWidth, m.listHeight)
		viewStr := m.renderEmailViewer(styles, m.mainWidth, m.viewHeight)

		// Border rendering based on focus
		listBorderColor := styles.PaneInactive
		if m.activePane == listPane {
			listBorderColor = styles.PaneActive
		}

		viewBorderColor := styles.PaneInactive
		if m.activePane == viewPane {
			viewBorderColor = styles.PaneActive
		}

		mainStr = lipgloss.JoinVertical(
			lipgloss.Left,
			listBorderColor.Width(m.mainWidth - 2).Height(m.listHeight - 2).Render(listStr),
			viewBorderColor.Width(m.mainWidth - 2).Height(m.viewHeight - 2).Render(viewStr),
		)
	}

	sidebarBorderColor := styles.PaneInactive
	if m.activePane == sidebarPane {
		sidebarBorderColor = styles.PaneActive
	}

	contentLayout := lipgloss.JoinHorizontal(
		lipgloss.Top,
		sidebarBorderColor.Width(sidebarWidth - 2).Height(m.contentHeight - 2).Render(sidebarStr),
		mainStr,
	)

	// Status bar at the very bottom
	statusBarStr := m.renderStatusBar(styles, appWidth)

	appLayout := lipgloss.JoinVertical(
		lipgloss.Left,
		contentLayout,
		statusBarStr,
	)

	if m.width > 130 {
		return lipgloss.PlaceHorizontal(m.width, lipgloss.Center, appLayout)
	}
	return appLayout
}

func (m model) renderSidebar(styles UIStyles, width, height int) string {
	var builder strings.Builder
	builder.WriteString(styles.SidebarTitle.Render("⚡ KANKERMAIL"))

	// Display connection indicator at sidebar top
	connectionIndicator := styles.StatusHelp.Render("Mode: Local Mock")
	if !m.config.UseMock {
		connectionIndicator = styles.ListItemUnread.Render("Mode: Server Sync")
	}
	builder.WriteString(" " + connectionIndicator + "\n\n")

	// Render first 6 folders (Inbox, Starred, Sent, Drafts, Archive, Trash)
	var topFolders []string
	for i := 0; i < len(folders)-1; i++ {
		f := folders[i]
		displayName := f
		switch f {
		case "inbox":
			displayName = "📥 Inbox"
		case "starred":
			displayName = "⭐ Starred"
		case "sent":
			displayName = "📤 Sent"
		case "drafts":
			displayName = "📝 Drafts"
		case "archive":
			displayName = "📦 Archive"
		case "trash":
			displayName = "🗑️ Trash"
		}

		// Calculate folder stats
		var stats string
		unreadCount := 0
		
		// Source list depends on mode
		var listToScan []Email
		if m.config.UseMock {
			listToScan = m.mailStore.Emails
		} else {
			listToScan = m.emails
		}

		for _, email := range listToScan {
			if f == "starred" {
				if email.Starred && email.Folder != "trash" {
					unreadCount++
				}
			} else if email.Folder == f {
				if email.Unread {
					unreadCount++
				}
			}
		}

		if unreadCount > 0 {
			stats = fmt.Sprintf(" (%d)", unreadCount)
		}

		itemStr := fmt.Sprintf("%s%s", displayName, stats)
		var renderedItem string
		if i == m.selectedFolderIdx {
			renderedItem = styles.SidebarItemSel.Width(width - 2).Render(itemStr)
		} else {
			renderedItem = styles.SidebarItem.Render(itemStr)
		}
		topFolders = append(topFolders, renderedItem)
	}

	// Output top folders
	for _, tf := range topFolders {
		builder.WriteString(tf + "\n")
	}

	// Calculate vertical spacing to push Settings to the bottom
	// Inner height of the sidebar is height - 2 (due to border)
	// Title block: 4 lines
	// Indicator block: 3 lines
	// Top folders: 6 lines
	// Settings: 1 line
	// Total top lines = 13 lines
	// Blank lines needed = (height - 2) - 13 - 1 = height - 16
	blankLines := height - 16
	if blankLines < 0 {
		blankLines = 0
	}
	builder.WriteString(strings.Repeat("\n", blankLines))

	// Render Settings at the bottom
	settingsIdx := len(folders) - 1
	var settingsStr string
	if m.selectedFolderIdx == settingsIdx {
		settingsStr = styles.SidebarItemSel.Width(width - 2).Render("⚙️ Settings")
	} else {
		settingsStr = styles.SidebarItem.Render("⚙️ Settings")
	}
	builder.WriteString(settingsStr + "\n")

	return builder.String()
}

func (m model) renderEmailList(styles UIStyles, width, height int) string {
	emails := m.getActiveEmails()

	var builder strings.Builder

	innerWidth := width - 2
	if innerWidth < 30 {
		innerWidth = 30
	}

	// Header row - subtract 2 for left/right padding inside styles.ListHeader
	headerStyle := styles.ListHeader.Width(innerWidth - 2)
	subjectWidth := innerWidth - 38
	if subjectWidth < 20 {
		subjectWidth = 20
	}
	headerRow := fmt.Sprintf(" %-22s %-*s %-12s", "FROM", subjectWidth, "SUBJECT", "DATE")
	headerRow = truncate.String(headerRow, uint(innerWidth))
	builder.WriteString(headerStyle.Render(headerRow) + "\n")

	if m.loading {
		builder.WriteString("\n  🔄 Syncing with IMAP server... Please wait...\n")
		return builder.String()
	}

	if len(emails) == 0 {
		builder.WriteString("\n  No messages in this folder.\n")
		return builder.String()
	}

	// Calculate visible list range (account for header and border bounds)
	pageSize := height - 3
	if pageSize < 1 {
		pageSize = 1
	}

	for i := m.listOffset; i < m.listOffset+pageSize && i < len(emails); i++ {
		email := emails[i]

		statusIcon := " "
		if email.Starred {
			statusIcon = "⭐"
		} else if email.Unread {
			statusIcon = "•"
		}

		from := email.From
		if len(from) > 20 {
			from = from[:17] + "..."
		}

		subject := email.Subject
		if len(subject) > subjectWidth {
			subject = subject[:subjectWidth-3] + "..."
		}

		dateStr := email.Date.Format("Jan _2 15:04")

		row := fmt.Sprintf(" %-2s %-20s %-*s %-12s", statusIcon, from, subjectWidth, subject, dateStr)
		row = truncate.String(row, uint(innerWidth))

		itemStyle := styles.ListItem
		if email.Unread {
			itemStyle = styles.ListItemUnread
		}

		if i == m.selectedEmailIdx {
			builder.WriteString(styles.ListItemSel.Width(innerWidth).Render(row) + "\n")
		} else {
			builder.WriteString(itemStyle.Render(row) + "\n")
		}
	}

	return strings.TrimRight(builder.String(), "\n")
}

func (m model) renderEmailViewer(styles UIStyles, width, height int) string {
	if m.loading {
		return "\n  🔄 Syncing..."
	}
	emails := m.getActiveEmails()
	if len(emails) == 0 {
		return "\n  Select a message to view content."
	}
	return m.viewport.View()
}

func (m model) renderSettings(styles UIStyles, width, height int) string {
	var builder strings.Builder

	borderStyle := styles.PaneInactive
	if m.activePane == listPane {
		borderStyle = styles.PaneActive
	}

	builder.WriteString(styles.ViewTitle.Render("⚙️ Application Settings") + "\n\n")

	for i := range settingsItems {
		indicator := "  "
		if i == m.selectedSettingsIdx {
			indicator = "➔ "
		}

		var label string
		if i == 0 {
			modeStatus := "Disabled (Sync Enabled)"
			if m.config.UseMock {
				modeStatus = "Enabled (Simulator Active)"
			}
			label = fmt.Sprintf("%s Mock Mode: %s", indicator, modeStatus)
		} else if i == 1 {
			htmlStatus := "Plain Text fallback"
			if m.config.HTMLRich {
				htmlStatus = "Rich HTML (Glamour styled)"
			}
			label = fmt.Sprintf("%s HTML Email Rendering: %s", indicator, htmlStatus)
		} else {
			themeIdx := i - 2
			t := AllThemes[themeIdx]
			label = fmt.Sprintf("%s Theme Preset: %s", indicator, t.Name)
			if t.Name == CurrentTheme.Name {
				label += " (Active)"
			}
		}

		lineStyle := styles.ListItem
		if i == m.selectedSettingsIdx {
			lineStyle = styles.ListItemSel.Width(width - 2)
		} else if i > 1 && AllThemes[i-2].Name == CurrentTheme.Name {
			lineStyle = styles.ListItemUnread
		}

		builder.WriteString(lineStyle.Render(label) + "\n")
	}

	builder.WriteString("\n" + strings.Repeat("─", width-2) + "\n\n")
	builder.WriteString(styles.ViewHeaderLabel.Render("Config File:") + " " + styles.ViewHeaderValue.Render(GetConfigPath()) + "\n")
	builder.WriteString(styles.ViewHeaderLabel.Render("IMAP Host:") + " " + styles.ViewHeaderValue.Render(m.config.IMAP.Host) + "\n")
	builder.WriteString(styles.ViewHeaderLabel.Render("IMAP Port:") + " " + styles.ViewHeaderValue.Render(fmt.Sprintf("%d", m.config.IMAP.Port)) + "\n")
	builder.WriteString(styles.ViewHeaderLabel.Render("IMAP User:") + " " + styles.ViewHeaderValue.Render(m.config.IMAP.Username) + "\n\n")
	
	builder.WriteString(styles.ViewHeaderLabel.Render("SMTP Host:") + " " + styles.ViewHeaderValue.Render(m.config.SMTP.Host) + "\n")
	builder.WriteString(styles.ViewHeaderLabel.Render("SMTP Port:") + " " + styles.ViewHeaderValue.Render(fmt.Sprintf("%d", m.config.SMTP.Port)) + "\n")
	builder.WriteString(styles.ViewHeaderLabel.Render("SMTP User:") + " " + styles.ViewHeaderValue.Render(m.config.SMTP.Username) + "\n\n")

	builder.WriteString("💡 " + styles.ListItemUnread.Render("Outlook (Office 365) Settings Guide:") + "\n")
	builder.WriteString("  - IMAP Host: " + styles.ViewHeaderValue.Render("outlook.office365.com") + " | Port: " + styles.ViewHeaderValue.Render("993") + " | SSL: " + styles.ViewHeaderValue.Render("true") + "\n")
	builder.WriteString("  - SMTP Host: " + styles.ViewHeaderValue.Render("smtp.office365.com") + " | Port: " + styles.ViewHeaderValue.Render("587") + " | SSL: " + styles.ViewHeaderValue.Render("false") + " (uses STARTTLS)\n")
	builder.WriteString("  - " + styles.StatusHelp.Render("Outlook requires an App Password! Generate one in your Microsoft account security page.") + "\n")

	return borderStyle.Width(width - 2).Height(height - 2).Render(strings.TrimRight(builder.String(), "\n"))
}

func (m model) renderStatusBar(styles UIStyles, width int) string {
	leftText := m.statusMessage
	if m.searching {
		leftText = "🔍 " + m.searchInput.View()
	}

	var helpText string
	if m.searching {
		helpText = "[Enter] Lock • [Esc] Cancel search"
	} else {
		folder := folders[m.selectedFolderIdx]
		switch m.activePane {
		case sidebarPane:
			helpText = "[↑/↓] Navigate • [Enter] Open • [Ctrl+R] Sync • [q] Quit"
		case listPane:
			if folder == "settings" {
				helpText = "[↑/↓] Select • [Enter] Save • [Tab] Back • [q] Quit"
			} else {
				helpText = "[↑/↓] List • [Enter] Read • [s] Search • [n] New • [d] Del"
			}
		case viewPane:
			helpText = "[↑/↓] Scroll • [Tab] List • [r] Reply • [d] Del • [esc] Back"
		}
	}

	left := styles.StatusMsg.Render(leftText)
	right := styles.StatusHelp.Render(helpText)

	maxWidth := width - 2
	if maxWidth < 20 {
		maxWidth = 20
	}

	// Prevent status bar from wrapping on smaller terminals
	if lipgloss.Width(left)+lipgloss.Width(right) > maxWidth {
		// Try to shorten the left status text first
		availableSpaceForLeft := maxWidth - lipgloss.Width(right) - 4
		if availableSpaceForLeft > 10 {
			leftText = leftText[:availableSpaceForLeft] + "..."
			left = styles.StatusMsg.Render(leftText)
		} else {
			// If still too small, drop the help text and only show status message
			right = ""
			leftTextWidth := lipgloss.Width(styles.StatusMsg.Render(leftText))
			if leftTextWidth > maxWidth {
				leftText = leftText[:maxWidth-4] + "..."
				left = styles.StatusMsg.Render(leftText)
			}
		}
	}

	spacing := (width - 1) - lipgloss.Width(left) - lipgloss.Width(right)
	if spacing < 0 {
		spacing = 0
	}

	return styles.StatusBar.Render(
		left + strings.Repeat(" ", spacing) + right,
	)
}

func main() {
	webMode := flag.Bool("web", false, "Start KankerMail in Web UI server mode")
	port := flag.Int("port", 8080, "Port to run the Web UI server on")
	flag.Parse()

	if *webMode {
		startWebServer(*port)
	} else {
		p := tea.NewProgram(initialModel(), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error running KankerMail: %v\n", err)
		}
	}
}
