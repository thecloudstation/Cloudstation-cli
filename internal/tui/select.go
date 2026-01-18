// Package tui provides terminal user interface components using Bubble Tea framework.
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// SelectModel represents a selection menu with arrow-key navigation.
type SelectModel struct {
	choices  []string
	cursor   int
	selected int
	done     bool
}

// NewSelectModel creates a new selection menu with the given choices.
func NewSelectModel(choices []string) SelectModel {
	return SelectModel{
		choices:  choices,
		cursor:   0,
		selected: -1,
		done:     false,
	}
}

// Init initializes the model. Required by tea.Model interface.
func (m SelectModel) Init() tea.Cmd {
	return nil
}

// Update handles incoming events and updates the model state.
// Required by tea.Model interface.
func (m SelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter":
			m.selected = m.cursor
			m.done = true
			return m, tea.Quit
		case "q", "esc", "ctrl+c":
			m.selected = -1
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

// View renders the current state of the model.
// Required by tea.Model interface.
func (m SelectModel) View() string {
	if m.done {
		return ""
	}

	s := "Select an option:\n\n"

	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
			s += SelectedStyle.Render(fmt.Sprintf("%s %s", cursor, choice)) + "\n"
		} else {
			s += fmt.Sprintf("%s %s\n", cursor, choice)
		}
	}

	s += "\n(up/down to move, Enter to select, q to quit)\n"

	return s
}

// Selected returns the index of the selected choice, or -1 if cancelled.
func (m SelectModel) Selected() int {
	return m.selected
}

// SelectedValue returns the selected choice string, or empty string if cancelled.
func (m SelectModel) SelectedValue() string {
	if m.selected >= 0 && m.selected < len(m.choices) {
		return m.choices[m.selected]
	}
	return ""
}

// IsDone returns true if the selection is complete (either selected or cancelled).
func (m SelectModel) IsDone() bool {
	return m.done
}

// RunSelect runs the selection menu and returns the selected index.
// Returns -1 if the user cancelled the selection.
func RunSelect(choices []string) (int, error) {
	if len(choices) == 0 {
		return -1, fmt.Errorf("no choices provided")
	}

	model := NewSelectModel(choices)
	p := tea.NewProgram(model)

	finalModel, err := p.Run()
	if err != nil {
		return -1, fmt.Errorf("failed to run select menu: %w", err)
	}

	m, ok := finalModel.(SelectModel)
	if !ok {
		return -1, fmt.Errorf("unexpected model type")
	}

	return m.Selected(), nil
}

// RunSelectWithTitle runs the selection menu with a custom title and returns the selected index.
// Returns -1 if the user cancelled the selection.
func RunSelectWithTitle(title string, choices []string) (int, error) {
	if len(choices) == 0 {
		return -1, fmt.Errorf("no choices provided")
	}

	model := NewSelectModelWithTitle(title, choices)
	p := tea.NewProgram(model)

	finalModel, err := p.Run()
	if err != nil {
		return -1, fmt.Errorf("failed to run select menu: %w", err)
	}

	m, ok := finalModel.(SelectModelWithTitle)
	if !ok {
		return -1, fmt.Errorf("unexpected model type")
	}

	return m.Selected(), nil
}

// SelectModelWithTitle is a SelectModel variant with a custom title.
type SelectModelWithTitle struct {
	SelectModel
	title string
}

// NewSelectModelWithTitle creates a new selection menu with a custom title.
func NewSelectModelWithTitle(title string, choices []string) SelectModelWithTitle {
	return SelectModelWithTitle{
		SelectModel: NewSelectModel(choices),
		title:       title,
	}
}

// Update handles incoming events and updates the model state.
func (m SelectModelWithTitle) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	updatedModel, cmd := m.SelectModel.Update(msg)
	m.SelectModel = updatedModel.(SelectModel)
	return m, cmd
}

// View renders the current state of the model with the custom title.
func (m SelectModelWithTitle) View() string {
	if m.done {
		return ""
	}

	s := m.title + "\n\n"

	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
			s += SelectedStyle.Render(fmt.Sprintf("%s %s", cursor, choice)) + "\n"
		} else {
			s += fmt.Sprintf("%s %s\n", cursor, choice)
		}
	}

	s += "\n(up/down to move, Enter to select, q to quit)\n"

	return s
}
