package styles

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	ColorPrimary   = lipgloss.Color("32")  // Green
	ColorSecondary = lipgloss.Color("212") // Pink
	ColorError     = lipgloss.Color("9")   // Red
	ColorSubtle    = lipgloss.Color("241") // Gray
	ColorHighlight = lipgloss.Color("39")  // Cyan

	// Core Styles
	BaseStyle = lipgloss.NewStyle().
			PaddingLeft(1).
			PaddingRight(1).
			PaddingTop(1).
			PaddingBottom(1)

	TitleStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true).
			MarginBottom(1)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true)

	SubtleStyle = lipgloss.NewStyle().
			Foreground(ColorSubtle)

	HighlightStyle = lipgloss.NewStyle().
			Foreground(ColorHighlight).
			Bold(true)

	// List Styles
	ListItemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	ListItemSelectedStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			Foreground(ColorHighlight)

	CheckboxCheckedStyle = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true)

	CheckboxUncheckedStyle = lipgloss.NewStyle().
				Foreground(ColorSubtle)

	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorSubtle).
			MarginTop(1)
)
