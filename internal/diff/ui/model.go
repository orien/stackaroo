/*
Copyright © 2025 Stackaroo Contributors
SPDX-License-Identifier: BSD-3-Clause
*/
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/v2/help"
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/viewport"
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

	// Bubbles components
	viewport viewport.Model
	help     help.Model

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
	styles        *diff.Styles
	keys          keyMap
	viewportKeys  viewport.KeyMap
}

// NewModel creates a new interactive diff viewer model
func NewModel(result *diff.Result, mode ViewMode) Model {
	useColour := diff.ShouldUseColour()

	m := Model{
		result:       result,
		sections:     buildSections(result),
		mode:         mode,
		useColour:    useColour,
		styles:       diff.NewStyles(useColour),
		keys:         defaultKeyMap(mode == Confirmation),
		viewportKeys: viewport.DefaultKeyMap(),
		help:         help.New(),
	}

	return m
}

// Init initialises the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle quit/cancel
		if key.Matches(msg, m.keys.Quit) {
			m.quitting = true
			if m.mode == Confirmation {
				m.cancelled = true
			}
			return m, tea.Quit
		}

		// Handle help toggle
		if key.Matches(msg, m.keys.Help) {
			m.showHelp = !m.showHelp
			m.help.ShowAll = m.showHelp
			return m, nil
		}

		// Handle confirmation mode actions
		if m.mode == Confirmation {
			if key.Matches(msg, m.keys.Confirm) {
				m.confirmed = true
				m.quitting = true
				return m, tea.Quit
			}
			if key.Matches(msg, m.keys.Cancel) {
				m.cancelled = true
				m.quitting = true
				return m, tea.Quit
			}
		}

		// Section navigation
		if key.Matches(msg, m.keys.NextSection) {
			m.nextSection()
			return m, nil
		}
		if key.Matches(msg, m.keys.PrevSection) {
			m.prevSection()
			return m, nil
		}

		// Delegate all other key handling to viewport (scrolling, etc.)
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		headerHeight := m.getHeaderHeight()
		footerHeight := m.getFooterHeight()
		verticalMarginHeight := headerHeight + footerHeight

		if !m.ready {
			m.viewport = viewport.New()
			m.viewport.SetWidth(msg.Width)
			m.viewport.SetHeight(msg.Height - verticalMarginHeight)
			m.viewport.YPosition = headerHeight
			m.ready = true
			m.updateContent()
		} else {
			m.viewport.SetWidth(msg.Width)
			m.viewport.SetHeight(msg.Height - verticalMarginHeight)
			m.viewport.YPosition = headerHeight
			m.updateContent()
		}

		// Let viewport handle the size message for its internal state
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
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
	s.WriteString(m.viewport.View())
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

// renderShortHelp renders short help hints using bubbles help component
func (m Model) renderShortHelp() string {
	m.help.ShowAll = false
	// Combine our keys with viewport keys for help display
	return m.help.View(combinedKeyMap{app: m.keys, viewport: m.viewportKeys})
}

// renderConfirmationPrompt renders a minimal inline confirmation prompt
func (m Model) renderConfirmationPrompt() string {
	// Build compact change summary
	summary := m.buildChangeSummary()

	// Create inline prompt with safe default (Enter = cancel)
	questionMark := m.styles.Modified.Render("?")
	promptText := lipgloss.NewStyle().Bold(true).Render("Deploy these changes?")
	acceptKeys := m.styles.Success.Render("y")
	cancelKeys := m.styles.Subtle.Render("n/enter/esc")

	promptLine := fmt.Sprintf("%s %s %s to deploy, %s to cancel", questionMark, promptText, acceptKeys, cancelKeys)

	// Navigation hints on second line
	hints := m.styles.Subtle.Render("↑↓/jk: scroll  •  tab: sections  •  ?: help")

	footer := lipgloss.JoinVertical(
		lipgloss.Left,
		summary,
		promptLine,
		hints,
	)

	return m.styles.Footer.Render(footer)
}

// buildChangeSummary creates a compact summary of changes for inline display
func (m Model) buildChangeSummary() string {
	if !m.result.StackExists {
		return m.styles.StatusNew.Render("New stack")
	}

	var parts []string

	// Resource changes (compact format)
	if m.result.TemplateChange != nil && m.result.TemplateChange.HasChanges {
		rc := m.result.TemplateChange.ResourceCount
		if rc.Added > 0 || rc.Modified > 0 || rc.Removed > 0 {
			parts = append(parts,
				m.styles.Added.Render(fmt.Sprintf("+%d", rc.Added)),
				m.styles.Modified.Render(fmt.Sprintf("~%d", rc.Modified)),
				m.styles.Removed.Render(fmt.Sprintf("-%d", rc.Removed)),
			)
		}
	}

	// Parameters and tags (compact)
	if len(m.result.ParameterDiffs) > 0 {
		parts = append(parts, fmt.Sprintf("%dp", len(m.result.ParameterDiffs)))
	}
	if len(m.result.TagDiffs) > 0 {
		parts = append(parts, fmt.Sprintf("%dt", len(m.result.TagDiffs)))
	}

	if len(parts) == 0 {
		return m.styles.Subtle.Render("No changes")
	}

	return strings.Join(parts, " ")
}

// renderFullHelp renders the full help view using bubbles help component
func (m Model) renderFullHelp() string {
	return m.help.View(combinedKeyMap{app: m.keys, viewport: m.viewportKeys})
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
	content := m.renderContent()
	m.viewport.SetContent(content)
}

// renderContent renders the main diff content
func (m Model) renderContent() string {
	var s strings.Builder

	// Show all sections
	for i, section := range m.sections {
		// Add visual separator between sections
		if i > 0 {
			s.WriteString("\n")
			s.WriteString(m.styles.Separator.Render(strings.Repeat("─", m.viewport.Width())))
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
		// Scroll to section start line
		m.viewport.SetYOffset(section.StartLine)
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
		return 3 // Minimal inline confirmation prompt (summary + prompt + hints)
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

// combinedKeyMap combines our application keys with viewport keys for help display
type combinedKeyMap struct {
	app      keyMap
	viewport viewport.KeyMap
}

// ShortHelp returns keybindings for short help
func (c combinedKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		c.viewport.Up,
		c.viewport.Down,
		c.app.NextSection,
		c.app.Help,
		c.app.Quit,
	}
}

// FullHelp returns keybindings for full help
func (c combinedKeyMap) FullHelp() [][]key.Binding {
	viewportKeys := []key.Binding{
		c.viewport.Up,
		c.viewport.Down,
		c.viewport.PageUp,
		c.viewport.PageDown,
		c.viewport.HalfPageUp,
		c.viewport.HalfPageDown,
	}

	appKeys := c.app.FullHelp()

	// Combine viewport keys with app keys
	result := [][]key.Binding{viewportKeys}
	result = append(result, appKeys...)

	return result
}
