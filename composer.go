package main

import (
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Composer represents the email creation form
type Composer struct {
	toInput      textinput.Model
	subjectInput textinput.Model
	bodyInput    textarea.Model
	focusIndex   int // 0 = To, 1 = Subject, 2 = Body
	width        int
	height       int
}

// NewComposer creates a new email composer
func NewComposer() Composer {
	to := textinput.New()
	to.Placeholder = "recipient@example.com"
	to.Focus()
	to.CharLimit = 156
	to.Width = 50

	subj := textinput.New()
	subj.Placeholder = "Enter subject line"
	subj.CharLimit = 156
	subj.Width = 50

	body := textarea.New()
	body.Placeholder = "Write your email here..."
	body.SetHeight(12)
	body.SetWidth(60)

	return Composer{
		toInput:      to,
		subjectInput: subj,
		bodyInput:    body,
		focusIndex:   0,
	}
}

// Reset clears all fields and resets focus
func (c *Composer) Reset() {
	c.toInput.SetValue("")
	c.subjectInput.SetValue("")
	c.bodyInput.SetValue("")
	c.focusIndex = 0
	c.toInput.Focus()
	c.subjectInput.Blur()
	c.bodyInput.Blur()
}

// GetEmail returns the compiled Email object from the composer values
func (c Composer) GetEmail() Email {
	return Email{
		From:    "you@example.com",
		To:      c.toInput.Value(),
		Subject: c.subjectInput.Value(),
		Body:    c.bodyInput.Value(),
		Folder:  "sent",
	}
}

// Init implements tea.Model
func (c Composer) Init() tea.Cmd {
	return nil
}

// Update handles message updates for the composer
func (c Composer) Update(msg tea.Msg) (Composer, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			c.focusIndex = (c.focusIndex + 1) % 3
			c.updateFocus()
			return c, nil
		case "shift+tab":
			c.focusIndex = (c.focusIndex - 1 + 3) % 3
			c.updateFocus()
			return c, nil
		}
	}

	// Update focused input
	switch c.focusIndex {
	case 0:
		c.toInput, cmd = c.toInput.Update(msg)
		cmds = append(cmds, cmd)
	case 1:
		c.subjectInput, cmd = c.subjectInput.Update(msg)
		cmds = append(cmds, cmd)
	case 2:
		c.bodyInput, cmd = c.bodyInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return c, tea.Batch(cmds...)
}

func (c *Composer) updateFocus() {
	if c.focusIndex == 0 {
		c.toInput.Focus()
	} else {
		c.toInput.Blur()
	}

	if c.focusIndex == 1 {
		c.subjectInput.Focus()
	} else {
		c.subjectInput.Blur()
	}

	if c.focusIndex == 2 {
		c.bodyInput.Focus()
	} else {
		c.bodyInput.Blur()
	}
}

// SetSize updates the size of the composer component
func (c *Composer) SetSize(width, height int) {
	c.width = width
	c.height = height
	c.toInput.Width = width - 20
	c.subjectInput.Width = width - 20
	c.bodyInput.SetWidth(width - 8)
	
	bodyHeight := height - 16
	if bodyHeight < 3 {
		bodyHeight = 3
	}
	c.bodyInput.SetHeight(bodyHeight)
}

// View renders the composer UI
func (c Composer) View(styles UIStyles) string {
	toView := c.toInput.View()
	subjView := c.subjectInput.View()
	bodyView := c.bodyInput.View()

	// Style inputs based on focus
	toLabel := styles.ComposerLabel.Render("To:")
	subjLabel := styles.ComposerLabel.Render("Subject:")
	bodyLabel := styles.ComposerLabel.Render("Message:")

	if c.focusIndex == 0 {
		toLabel = styles.ComposerLabel.Copy().Foreground(lipgloss.Color(CurrentTheme.Success)).Render("To:")
	} else if c.focusIndex == 1 {
		subjLabel = styles.ComposerLabel.Copy().Foreground(lipgloss.Color(CurrentTheme.Success)).Render("Subject:")
	} else if c.focusIndex == 2 {
		bodyLabel = styles.ComposerLabel.Copy().Foreground(lipgloss.Color(CurrentTheme.Success)).Render("Message:")
	}

	form := lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Center, toLabel, toView),
		"",
		lipgloss.JoinHorizontal(lipgloss.Center, subjLabel, subjView),
		"",
		bodyLabel,
		bodyView,
		"",
		styles.StatusHelp.Render("Press [Tab] to cycle inputs • [Ctrl+S] Send • [Esc] Cancel"),
	)

	modalHeight := c.height - 4
	if modalHeight < 12 {
		modalHeight = 12
	}

	return styles.ModalBorder.
		Width(c.width - 6).
		Height(modalHeight).
		Render(form)
}
