// Package tui provides terminal user interface components for the CloudStation CLI.
// It includes consistent theming, styling, and reusable UI elements.
package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Color palette - CloudStation brand theme (Orange, Black, White)
var (
	// Primary brand color - Orange #ff8700
	ColorPrimary = lipgloss.Color("208")

	// Secondary brand color - Light Orange #ffaf00
	ColorSecondary = lipgloss.Color("214")

	// Accent color - Dark Orange #ff5f00
	ColorAccent = lipgloss.Color("202")

	// Black for backgrounds #000000
	ColorBlack = lipgloss.Color("16")

	// White for high contrast text #eeeeee
	ColorWhite = lipgloss.Color("255")

	// Gray for secondary text #585858
	ColorGray = lipgloss.Color("240")

	// Dark Gray for subtle backgrounds #303030
	ColorDarkGray = lipgloss.Color("236")

	// Success indicator - Green
	ColorSuccess = lipgloss.Color("42")

	// Error indicator - Red
	ColorError = lipgloss.Color("196")

	// Warning indicator - Orange (same as secondary for brand consistency)
	ColorWarning = lipgloss.Color("214")

	// Muted/subtle text - Gray
	ColorMuted = lipgloss.Color("240")
)

// Text styles - reusable styles for common text elements
var (
	// TitleStyle is used for main titles and headings
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			MarginBottom(1)

	// HeaderStyle is used for section headers
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSecondary)

	// SubheaderStyle is used for sub-section headers
	SubheaderStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Italic(true)

	// SelectedStyle is used for selected/active items
	SelectedStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	// ErrorStyle is used for error messages
	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true)

	// SuccessStyle is used for success messages
	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true)

	// WarningStyle is used for warning messages
	WarningStyle = lipgloss.NewStyle().
			Foreground(ColorWarning)

	// MutedStyle is used for subtle/secondary text
	MutedStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// HighlightStyle is used for highlighted/emphasized text
	HighlightStyle = lipgloss.NewStyle().
			Background(ColorPrimary).
			Foreground(ColorBlack).
			Padding(0, 1)

	// LinkStyle is used for links and clickable elements
	LinkStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Underline(true)

	// BoldStyle is used for bold text without color
	BoldStyle = lipgloss.NewStyle().
			Bold(true)

	// DimStyle is used for dimmed/disabled text
	DimStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Faint(true)

	// BrandStyle is the CloudStation brand style - orange on black
	BrandStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Background(ColorBlack).
			Bold(true).
			Padding(0, 1)
)

// Box styles - reusable styles for container elements
var (
	// BoxStyle is the default box/container style
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(1, 2)

	// FocusedBoxStyle is used for focused/active containers
	FocusedBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorSecondary).
			Padding(1, 2)

	// ErrorBoxStyle is used for error message containers
	ErrorBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorError).
			Padding(1, 2)

	// SuccessBoxStyle is used for success message containers
	SuccessBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorSuccess).
			Padding(1, 2)

	// WarningBoxStyle is used for warning message containers
	WarningBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorWarning).
			Padding(1, 2)
)

// List styles - styles for list/menu components
var (
	// ListItemStyle is the default style for list items
	ListItemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	// ActiveListItemStyle is used for the currently selected list item
	ActiveListItemStyle = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true).
				PaddingLeft(2)

	// ActiveItemStyle is an alias for ActiveListItemStyle (for compatibility)
	ActiveItemStyle = ActiveListItemStyle

	// ListCursorStyle is used for the selection cursor/indicator
	ListCursorStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	// CursorStyle is an alias for ListCursorStyle (for compatibility)
	CursorStyle = ListCursorStyle
)

// Input styles - styles for form inputs
var (
	// InputStyle is the default style for text inputs
	InputStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(ColorGray).
			Padding(0, 1)

	// FocusedInputStyle is used for focused text inputs
	FocusedInputStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(ColorPrimary).
				Padding(0, 1)

	// InputLabelStyle is used for input field labels
	InputLabelStyle = lipgloss.NewStyle().
			Foreground(ColorWhite).
			Bold(true)

	// InputPlaceholderStyle is used for placeholder text in inputs
	InputPlaceholderStyle = lipgloss.NewStyle().
				Foreground(ColorMuted)
)

// Progress styles - styles for progress indicators
var (
	// ProgressBarStyle is used for progress bar backgrounds
	ProgressBarStyle = lipgloss.NewStyle().
				Foreground(ColorMuted)

	// ProgressFilledStyle is used for the filled portion of progress bars
	ProgressFilledStyle = lipgloss.NewStyle().
				Foreground(ColorPrimary)

	// SpinnerStyle is used for loading spinners
	SpinnerStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary)
)

// Status indicators - character constants for status display
const (
	// StatusSuccess is the checkmark character for success
	StatusSuccess = "[OK]"

	// StatusError is the X character for errors
	StatusError = "[ERR]"

	// StatusWarning is the warning character
	StatusWarning = "[WARN]"

	// StatusInfo is the info character
	StatusInfo = "[INFO]"

	// StatusPending is the pending/loading character
	StatusPending = "[...]"

	// ListCursor is the cursor character for list selection
	ListCursor = ">"

	// ListBullet is the bullet character for unselected list items
	ListBullet = "-"
)

// RenderTitle renders text with the title style
func RenderTitle(text string) string {
	return TitleStyle.Render(text)
}

// RenderHeader renders text with the header style
func RenderHeader(text string) string {
	return HeaderStyle.Render(text)
}

// RenderSuccess renders a success message with checkmark
func RenderSuccess(text string) string {
	return SuccessStyle.Render(StatusSuccess + " " + text)
}

// RenderError renders an error message with X mark
func RenderError(text string) string {
	return ErrorStyle.Render(StatusError + " " + text)
}

// RenderWarning renders a warning message with warning symbol
func RenderWarning(text string) string {
	return WarningStyle.Render(StatusWarning + " " + text)
}

// RenderInfo renders an info message with info symbol
func RenderInfo(text string) string {
	return MutedStyle.Render(StatusInfo + " " + text)
}

// RenderMuted renders text with the muted style
func RenderMuted(text string) string {
	return MutedStyle.Render(text)
}

// RenderHighlight renders text with the highlight style
func RenderHighlight(text string) string {
	return HighlightStyle.Render(text)
}

// RenderSelected renders text with the selected style
func RenderSelected(text string) string {
	return SelectedStyle.Render(text)
}

// RenderLink renders text as a link
func RenderLink(text string) string {
	return LinkStyle.Render(text)
}

// RenderBrand renders text with the CloudStation brand style
func RenderBrand(text string) string {
	return BrandStyle.Render(text)
}

// RenderBox wraps content in a styled box
func RenderBox(content string) string {
	return BoxStyle.Render(content)
}

// RenderFocusedBox wraps content in a focused styled box
func RenderFocusedBox(content string) string {
	return FocusedBoxStyle.Render(content)
}

// RenderErrorBox wraps content in an error styled box
func RenderErrorBox(content string) string {
	return ErrorBoxStyle.Render(content)
}

// RenderSuccessBox wraps content in a success styled box
func RenderSuccessBox(content string) string {
	return SuccessBoxStyle.Render(content)
}

// RenderWarningBox wraps content in a warning styled box
func RenderWarningBox(content string) string {
	return WarningBoxStyle.Render(content)
}

// RenderListItem renders a list item with optional selection state
func RenderListItem(text string, selected bool) string {
	if selected {
		return ActiveListItemStyle.Render(ListCursor + " " + text)
	}
	return ListItemStyle.Render(ListBullet + " " + text)
}

// RenderStatusLine renders a status line with label and value
func RenderStatusLine(label, value string, status string) string {
	labelStyled := InputLabelStyle.Render(label + ":")
	var valueStyled string

	switch status {
	case "success":
		valueStyled = SuccessStyle.Render(value)
	case "error":
		valueStyled = ErrorStyle.Render(value)
	case "warning":
		valueStyled = WarningStyle.Render(value)
	default:
		valueStyled = value
	}

	return labelStyled + " " + valueStyled
}

// RenderLogo renders the CloudStation logo/brand header
func RenderLogo() string {
	return lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true).
		Render("CloudStation")
}

// Width returns a style with the specified width
func Width(style lipgloss.Style, width int) lipgloss.Style {
	return style.Width(width)
}

// Height returns a style with the specified height
func Height(style lipgloss.Style, height int) lipgloss.Style {
	return style.Height(height)
}

// Center returns a style that centers content horizontally
func Center(style lipgloss.Style) lipgloss.Style {
	return style.Align(lipgloss.Center)
}
