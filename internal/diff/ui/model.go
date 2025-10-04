/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/orien/stackaroo/internal/diff"
)

// ViewMode determines the behaviour of the diff viewer
type ViewMode int

const (
	// ViewOnly mode allows browsing the diff
	ViewOnly ViewMode = iota
	// Confirmation mode prompts for deployment confirmation
	Confirmation
)

// Section represents a navigable section in the diff
type Section struct {
	Name       string
	Content    string
	HasChanges bool
	StartLine  int // Line number where this section starts
}

// Model is the Bubble Tea model for the interactive diff viewer
type Model struct {
	// Core data
	result   *diff.Result
	sections []Section

	// Viewport state
	content        string   // Full rendered content
	contentLines   []string // Content split into lines
	yOffset        int      // Current scroll position
	viewportWidth  int
	viewportHeight int

	// State
	mode          ViewMode
	activeSection int
	showHelp      bool
	width         int
	height        int
	ready         bool
	quitting      bool
	confirmed     bool // User confirmed (for Confirmation mode)
	cancelled     bool // User cancelled (for Confirmation mode)
	useColour     bool
	styles        *StyleSet
	keys          KeyMap
}

// NewModel creates a new interactive diff viewer model
func NewModel(result *diff.Result, mode ViewMode) Model {
	useColour := shouldUseColour()

	return Model{
		result:    result,
		sections:  buildSections(result),
		mode:      mode,
		useColour: useColour,
		styles:    NewStyleSet(useColour),
		keys:      DefaultKeyMap(mode == Confirmation),
	}
}

// Init initialises the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle global keys first
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.quitting = true
			if m.mode == Confirmation {
				m.cancelled = true
			}
			return m, tea.Quit

		case "?":
			m.showHelp = !m.showHelp
			return m, nil

		case "enter", "y":
			if m.mode == Confirmation {
				m.confirmed = true
				m.quitting = true
				return m, tea.Quit
			}

		case "n":
			if m.mode == Confirmation {
				m.cancelled = true
				m.quitting = true
				return m, tea.Quit
			}

		// Section navigation
		case "tab", "right", "l":
			m.nextSection()
			return m, nil

		case "shift+tab", "left", "h":
			m.prevSection()
			return m, nil

		// Viewport scrolling
		case "up", "k":
			m.scrollUp(1)

		case "down", "j":
			m.scrollDown(1)

		case "pgup", "b":
			m.scrollUp(m.viewportHeight)

		case "pgdown", "f", " ":
			m.scrollDown(m.viewportHeight)

		case "u", "ctrl+u":
			m.scrollUp(m.viewportHeight / 2)

		case "d", "ctrl+d":
			m.scrollDown(m.viewportHeight / 2)

		case "home", "g":
			m.yOffset = 0

		case "end", "G":
			m.yOffset = max(0, len(m.contentLines)-m.viewportHeight)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if !m.ready {
			m.viewportHeight = msg.Height - m.getHeaderHeight() - m.getFooterHeight()
			m.viewportWidth = msg.Width
			m.ready = true
			m.updateContent()
		} else {
			m.viewportHeight = msg.Height - m.getHeaderHeight() - m.getFooterHeight()
			m.viewportWidth = msg.Width
			m.updateContent()
		}
	}

	return m, nil
}

// View renders the model
func (m Model) View() string {
	if !m.ready {
		return "Initialising..."
	}

	var s strings.Builder

	// Header
	s.WriteString(m.renderHeader())
	s.WriteString("\n")

	// Content (viewport)
	s.WriteString(m.renderViewport())
	s.WriteString("\n")

	// Footer
	s.WriteString(m.renderFooter())

	return s.String()
}

// renderHeader renders the header section
func (m Model) renderHeader() string {
	title := lipgloss.JoinHorizontal(
		lipgloss.Left,
		m.styles.HeaderTitle.Render("Stack: "),
		m.styles.HeaderValue.Render(m.result.StackName),
		m.styles.HeaderTitle.Render("  Context: "),
		m.styles.HeaderValue.Render(m.result.Context),
	)

	// Status indicator
	var status string
	if !m.result.StackExists {
		status = m.styles.StatusNew.Render("● NEW STACK")
	} else if m.result.HasChanges() {
		status = m.styles.StatusChanges.Render("● CHANGES DETECTED")
	} else {
		status = m.styles.StatusNoChange.Render("● NO CHANGES")
	}

	header := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		status,
	)

	return m.styles.Header.Render(header)
}

// renderViewport renders the visible portion of the content
func (m Model) renderViewport() string {
	if len(m.contentLines) == 0 {
		return ""
	}

	start := m.yOffset
	end := min(start+m.viewportHeight, len(m.contentLines))

	if start >= len(m.contentLines) {
		start = max(0, len(m.contentLines)-m.viewportHeight)
		end = len(m.contentLines)
	}

	visibleLines := m.contentLines[start:end]
	return strings.Join(visibleLines, "\n")
}

// renderFooter renders the footer section
func (m Model) renderFooter() string {
	if m.showHelp {
		return m.styles.Footer.Render(m.renderFullHelp())
	}

	// In confirmation mode, show prominent confirmation prompt
	if m.mode == Confirmation {
		return m.renderConfirmationPrompt()
	}

	// Show section navigation
	sectionInfo := m.renderSectionNav()

	// Show key hints
	hints := m.renderShortHelp()

	footer := lipgloss.JoinVertical(
		lipgloss.Left,
		sectionInfo,
		hints,
	)

	return m.styles.Footer.Render(footer)
}

// renderShortHelp renders short help hints
func (m Model) renderShortHelp() string {
	var hints []string

	hints = append(hints, "↑↓/jk: scroll")
	hints = append(hints, "tab/shift+tab: sections")
	hints = append(hints, "?: help")
	hints = append(hints, "q/esc: quit")

	return strings.Join(hints, "  •  ")
}

// renderConfirmationPrompt renders a prominent confirmation prompt
func (m Model) renderConfirmationPrompt() string {
	var s strings.Builder

	// Build change summary
	summary := m.buildChangeSummary()

	// Render prominent confirmation box
	promptStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("11")). // Yellow
		Padding(1, 2).
		Bold(true)

	prompt := lipgloss.JoinVertical(
		lipgloss.Left,
		"⚠  DEPLOYMENT CONFIRMATION",
		"",
		summary,
		"",
		"Press [Enter] or [Y] to deploy  •  Press [N] to cancel",
	)

	s.WriteString(promptStyle.Render(prompt))
	s.WriteString("\n\n")

	// Add navigation hints below
	hints := "↑↓/jk: scroll  •  tab: sections  •  ?: help"
	s.WriteString(m.styles.Subtle.Render(hints))

	return s.String()
}

// buildChangeSummary creates a summary of changes
func (m Model) buildChangeSummary() string {
	if !m.result.StackExists {
		return "Creating new stack: " + m.result.StackName
	}

	var parts []string

	if m.result.TemplateChange != nil && m.result.TemplateChange.HasChanges {
		rc := m.result.TemplateChange.ResourceCount
		if rc.Added > 0 || rc.Modified > 0 || rc.Removed > 0 {
			parts = append(parts, m.styles.Added.Render("+"+fmt.Sprintf("%d", rc.Added)))
			parts = append(parts, m.styles.Modified.Render("~"+fmt.Sprintf("%d", rc.Modified)))
			parts = append(parts, m.styles.Removed.Render("-"+fmt.Sprintf("%d", rc.Removed)))
		}
	}

	if len(m.result.ParameterDiffs) > 0 {
		parts = append(parts, fmt.Sprintf("%d parameter changes", len(m.result.ParameterDiffs)))
	}

	if len(m.result.TagDiffs) > 0 {
		parts = append(parts, fmt.Sprintf("%d tag changes", len(m.result.TagDiffs)))
	}

	if len(parts) == 0 {
		return "Updating stack: " + m.result.StackName
	}

	return "Changes: " + strings.Join(parts, " ")
}

// renderFullHelp renders the full help view
func (m Model) renderFullHelp() string {
	var s strings.Builder

	s.WriteString(m.styles.SectionHeader.Render("Keyboard Shortcuts"))
	s.WriteString("\n\n")

	s.WriteString("Navigation:\n")
	s.WriteString("  ↑/k       scroll up         ↓/j       scroll down\n")
	s.WriteString("  pgup/b    page up           pgdn/f    page down\n")
	s.WriteString("  u         half page up      d         half page down\n")
	s.WriteString("  g/home    go to top         G/end     go to bottom\n\n")

	s.WriteString("Sections:\n")
	s.WriteString("  tab/→/l   next section      shift+tab/←/h   previous section\n\n")

	if m.mode == Confirmation {
		s.WriteString("Actions:\n")
		s.WriteString("  enter/y   confirm & deploy  n         cancel\n")
		s.WriteString("  q/esc     cancel\n\n")
	} else {
		s.WriteString("Actions:\n")
		s.WriteString("  q/esc     quit              ?         toggle help\n\n")
	}

	return s.String()
}

// renderSectionNav renders the section navigation indicator
func (m Model) renderSectionNav() string {
	if len(m.sections) == 0 {
		return ""
	}

	var parts []string
	for i, section := range m.sections {
		style := m.styles.SectionInactive
		if i == m.activeSection {
			style = m.styles.SectionActive
		}

		indicator := ""
		if section.HasChanges {
			indicator = "● "
		}

		parts = append(parts, style.Render(indicator+section.Name))
	}

	return "Sections: " + lipgloss.JoinHorizontal(lipgloss.Left, parts...)
}

// updateContent updates the viewport content
func (m *Model) updateContent() {
	m.content = m.renderContent()
	m.contentLines = strings.Split(m.content, "\n")
}

// renderContent renders the main diff content
func (m Model) renderContent() string {
	var s strings.Builder

	// Show all sections
	for i, section := range m.sections {
		// Add visual separator between sections
		if i > 0 {
			s.WriteString("\n")
			s.WriteString(m.styles.Separator.Render(strings.Repeat("─", m.viewportWidth)))
			s.WriteString("\n\n")
		}

		// Highlight active section
		if i == m.activeSection {
			s.WriteString(m.styles.SectionHeader.Render("▶ " + section.Name))
		} else {
			s.WriteString(m.styles.SectionHeaderInactive.Render("  " + section.Name))
		}
		s.WriteString("\n")
		s.WriteString(m.styles.Separator.Render(strings.Repeat("─", len(section.Name)+2)))
		s.WriteString("\n\n")

		// Section content
		s.WriteString(section.Content)
		s.WriteString("\n")
	}

	return s.String()
}

// scrollUp scrolls the viewport up by n lines
func (m *Model) scrollUp(n int) {
	m.yOffset = max(0, m.yOffset-n)
}

// scrollDown scrolls the viewport down by n lines
func (m *Model) scrollDown(n int) {
	maxOffset := max(0, len(m.contentLines)-m.viewportHeight)
	m.yOffset = min(maxOffset, m.yOffset+n)
}

// nextSection moves to the next section
func (m *Model) nextSection() {
	if len(m.sections) == 0 {
		return
	}
	m.activeSection = (m.activeSection + 1) % len(m.sections)
	m.scrollToSection()
}

// prevSection moves to the previous section
func (m *Model) prevSection() {
	if len(m.sections) == 0 {
		return
	}
	m.activeSection--
	if m.activeSection < 0 {
		m.activeSection = len(m.sections) - 1
	}
	m.scrollToSection()
}

// scrollToSection scrolls the viewport to show the active section
func (m *Model) scrollToSection() {
	if m.activeSection >= 0 && m.activeSection < len(m.sections) {
		section := m.sections[m.activeSection]
		// Scroll to section start, but ensure we don't scroll past the end
		m.yOffset = min(section.StartLine, max(0, len(m.contentLines)-m.viewportHeight))
	}
}

// getHeaderHeight returns the height of the header
func (m Model) getHeaderHeight() int {
	return 3 // Title line + status line + spacing
}

// getFooterHeight returns the height of the footer
func (m Model) getFooterHeight() int {
	if m.showHelp {
		return 12 // Approximate height of full help
	}
	if m.mode == Confirmation {
		return 8 // Confirmation box height
	}
	return 3 // Section nav + help hints + spacing
}

// Confirmed returns true if the user confirmed (Confirmation mode only)
func (m Model) Confirmed() bool {
	return m.confirmed
}

// Cancelled returns true if the user cancelled (Confirmation mode only)
func (m Model) Cancelled() bool {
	return m.cancelled
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
