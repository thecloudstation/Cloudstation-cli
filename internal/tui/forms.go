// Package tui provides terminal user interface components for collecting user input.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Form-specific styles (using theme colors for consistency)
var (
	formFocusedStyle = lipgloss.NewStyle().Foreground(ColorSecondary)
	formBlurredStyle = lipgloss.NewStyle().Foreground(ColorMuted)
	formNoStyle      = lipgloss.NewStyle()
	formHelpStyle    = lipgloss.NewStyle().Foreground(ColorMuted)
	formErrorStyle   = lipgloss.NewStyle().Foreground(ColorError)
)

// ValidationFunc is a function that validates a field value.
// It returns an error message if validation fails, or an empty string if valid.
type ValidationFunc func(value string) string

// FormField represents a single input field in a form.
type FormField struct {
	// Label is the display label for the field
	Label string

	// Input is the underlying text input model
	Input textinput.Model

	// Required indicates if the field must have a value
	Required bool

	// Validate is an optional validation function for the field
	Validate ValidationFunc

	// validationError holds the current validation error message
	validationError string
}

// FormModel represents a form with multiple input fields.
type FormModel struct {
	// fields contains all form fields
	fields []FormField

	// focusIndex tracks which field is currently focused
	focusIndex int

	// done indicates if the form interaction is complete
	done bool

	// submitted indicates if the form was submitted (true) or cancelled (false)
	submitted bool

	// title is an optional title displayed at the top of the form
	title string

	// width is the form width for layout purposes
	width int
}

// FormOption is a function that configures a FormModel.
type FormOption func(*FormModel)

// WithTitle sets the form title.
func WithTitle(title string) FormOption {
	return func(m *FormModel) {
		m.title = title
	}
}

// WithWidth sets the form width.
func WithWidth(width int) FormOption {
	return func(m *FormModel) {
		m.width = width
	}
}

// WithFieldRequired marks a specific field as required.
func WithFieldRequired(index int, required bool) FormOption {
	return func(m *FormModel) {
		if index >= 0 && index < len(m.fields) {
			m.fields[index].Required = required
		}
	}
}

// WithFieldValidation sets a validation function for a specific field.
func WithFieldValidation(index int, validate ValidationFunc) FormOption {
	return func(m *FormModel) {
		if index >= 0 && index < len(m.fields) {
			m.fields[index].Validate = validate
		}
	}
}

// WithPlaceholder sets a placeholder for a specific field.
func WithPlaceholder(index int, placeholder string) FormOption {
	return func(m *FormModel) {
		if index >= 0 && index < len(m.fields) {
			m.fields[index].Input.Placeholder = placeholder
		}
	}
}

// WithDefaultValue sets a default value for a specific field.
func WithDefaultValue(index int, value string) FormOption {
	return func(m *FormModel) {
		if index >= 0 && index < len(m.fields) {
			m.fields[index].Input.SetValue(value)
		}
	}
}

// WithCharLimit sets the character limit for a specific field.
func WithCharLimit(index int, limit int) FormOption {
	return func(m *FormModel) {
		if index >= 0 && index < len(m.fields) {
			m.fields[index].Input.CharLimit = limit
		}
	}
}

// WithEchoMode sets the echo mode for a specific field (useful for passwords).
func WithEchoMode(index int, mode textinput.EchoMode) FormOption {
	return func(m *FormModel) {
		if index >= 0 && index < len(m.fields) {
			m.fields[index].Input.EchoMode = mode
		}
	}
}

// NewFormModel creates a new form model with the specified field labels.
func NewFormModel(labels []string, opts ...FormOption) FormModel {
	fields := make([]FormField, len(labels))

	for i, label := range labels {
		ti := textinput.New()
		ti.Placeholder = label
		ti.CharLimit = 156
		ti.Width = 40
		ti.PromptStyle = formFocusedStyle
		ti.TextStyle = formNoStyle

		// Focus first field
		if i == 0 {
			ti.Focus()
			ti.PromptStyle = formFocusedStyle
			ti.TextStyle = formNoStyle
		}

		fields[i] = FormField{
			Label:    label,
			Input:    ti,
			Required: false,
		}
	}

	m := FormModel{
		fields:     fields,
		focusIndex: 0,
		done:       false,
		submitted:  false,
		width:      60,
	}

	// Apply options
	for _, opt := range opts {
		opt(&m)
	}

	return m
}

// Init implements tea.Model.
func (m FormModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update implements tea.Model.
func (m FormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.done = true
			m.submitted = false
			return m, tea.Quit

		case "tab", "shift+tab", "enter", "up", "down":
			s := msg.String()

			// Validate current field before moving
			if m.focusIndex >= 0 && m.focusIndex < len(m.fields) {
				field := &m.fields[m.focusIndex]
				field.validationError = m.validateField(m.focusIndex)
			}

			// Submit on enter if on last field and validation passes
			if s == "enter" && m.focusIndex == len(m.fields)-1 {
				if m.validateAll() {
					m.done = true
					m.submitted = true
					return m, tea.Quit
				}
				// If validation failed, don't quit - just stay on the form
				return m, nil
			}

			// Navigate fields
			if s == "up" || s == "shift+tab" {
				m.focusIndex--
			} else {
				m.focusIndex++
			}

			// Wrap around
			if m.focusIndex > len(m.fields)-1 {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = len(m.fields) - 1
			}

			// Update focus state
			cmds := make([]tea.Cmd, len(m.fields))
			for i := 0; i < len(m.fields); i++ {
				if i == m.focusIndex {
					cmds[i] = m.fields[i].Input.Focus()
					m.fields[i].Input.PromptStyle = formFocusedStyle
					m.fields[i].Input.TextStyle = formNoStyle
				} else {
					m.fields[i].Input.Blur()
					m.fields[i].Input.PromptStyle = formNoStyle
					m.fields[i].Input.TextStyle = formNoStyle
				}
			}

			return m, tea.Batch(cmds...)
		}
	}

	// Update focused field
	cmd := m.updateInputs(msg)
	return m, cmd
}

// updateInputs updates all input fields with the given message.
func (m *FormModel) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.fields))

	for i := range m.fields {
		m.fields[i].Input, cmds[i] = m.fields[i].Input.Update(msg)
	}

	return tea.Batch(cmds...)
}

// validateField validates a single field and returns an error message or empty string.
func (m FormModel) validateField(index int) string {
	if index < 0 || index >= len(m.fields) {
		return ""
	}

	field := m.fields[index]
	value := field.Input.Value()

	// Check required
	if field.Required && strings.TrimSpace(value) == "" {
		return fmt.Sprintf("%s is required", field.Label)
	}

	// Run custom validation
	if field.Validate != nil {
		return field.Validate(value)
	}

	return ""
}

// validateAll validates all fields and returns true if all are valid.
func (m *FormModel) validateAll() bool {
	allValid := true

	for i := range m.fields {
		errMsg := m.validateField(i)
		m.fields[i].validationError = errMsg
		if errMsg != "" {
			allValid = false
		}
	}

	return allValid
}

// View implements tea.Model.
func (m FormModel) View() string {
	if m.done {
		return ""
	}

	var b strings.Builder

	// Render title if present
	if m.title != "" {
		titleStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99")).
			MarginBottom(1)
		b.WriteString(titleStyle.Render(m.title))
		b.WriteString("\n\n")
	}

	// Render each field
	for i, field := range m.fields {
		// Label with required indicator
		labelText := field.Label
		if field.Required {
			labelText += " *"
		}

		// Style label based on focus
		var labelStyle lipgloss.Style
		if i == m.focusIndex {
			labelStyle = formFocusedStyle.Bold(true)
		} else {
			labelStyle = formBlurredStyle
		}

		b.WriteString(labelStyle.Render(labelText))
		b.WriteString(":\n")

		// Input field
		b.WriteString(field.Input.View())
		b.WriteString("\n")

		// Validation error
		if field.validationError != "" {
			b.WriteString(formErrorStyle.Render("  " + field.validationError))
			b.WriteString("\n")
		}

		b.WriteString("\n")
	}

	// Help text
	b.WriteString(formHelpStyle.Render("(Tab to navigate, Enter to submit, Esc to cancel)"))
	b.WriteString("\n")

	return b.String()
}

// Values returns the form values if submitted, or nil if cancelled.
func (m FormModel) Values() []string {
	if !m.submitted {
		return nil
	}

	values := make([]string, len(m.fields))
	for i, field := range m.fields {
		values[i] = field.Input.Value()
	}
	return values
}

// ValueMap returns the form values as a map with labels as keys.
// Returns nil if the form was cancelled.
func (m FormModel) ValueMap() map[string]string {
	if !m.submitted {
		return nil
	}

	values := make(map[string]string)
	for _, field := range m.fields {
		values[field.Label] = field.Input.Value()
	}
	return values
}

// IsSubmitted returns true if the form was submitted.
func (m FormModel) IsSubmitted() bool {
	return m.submitted
}

// IsCancelled returns true if the form was cancelled.
func (m FormModel) IsCancelled() bool {
	return m.done && !m.submitted
}

// FieldCount returns the number of fields in the form.
func (m FormModel) FieldCount() int {
	return len(m.fields)
}

// FocusedIndex returns the index of the currently focused field.
func (m FormModel) FocusedIndex() int {
	return m.focusIndex
}

// RunForm displays a form with the given labels and returns the values.
// Returns nil if the form was cancelled.
func RunForm(labels []string, opts ...FormOption) ([]string, error) {
	model := NewFormModel(labels, opts...)
	p := tea.NewProgram(model)

	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to run form: %w", err)
	}

	m := finalModel.(FormModel)
	return m.Values(), nil
}

// RunFormWithMap displays a form and returns the values as a map.
// Returns nil if the form was cancelled.
func RunFormWithMap(labels []string, opts ...FormOption) (map[string]string, error) {
	model := NewFormModel(labels, opts...)
	p := tea.NewProgram(model)

	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to run form: %w", err)
	}

	m := finalModel.(FormModel)
	return m.ValueMap(), nil
}
