// Package tui provides terminal user interface components for the CloudStation CLI.
// It uses the bubbletea framework for building rich, interactive terminal applications.
package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SpinnerAnimation defines the visual animation type for spinners.
type SpinnerAnimation int

const (
	// SpinnerAnimationDot uses a simple dot animation (default).
	SpinnerAnimationDot SpinnerAnimation = iota
	// SpinnerAnimationLine uses a line-based animation.
	SpinnerAnimationLine
	// SpinnerAnimationMiniDot uses a compact dot animation.
	SpinnerAnimationMiniDot
	// SpinnerAnimationJump uses a jumping animation.
	SpinnerAnimationJump
	// SpinnerAnimationPulse uses a pulsing animation.
	SpinnerAnimationPulse
	// SpinnerAnimationPoints uses a points animation.
	SpinnerAnimationPoints
	// SpinnerAnimationGlobe uses a globe animation.
	SpinnerAnimationGlobe
	// SpinnerAnimationMoon uses a moon phases animation.
	SpinnerAnimationMoon
	// SpinnerAnimationMonkey uses a monkey animation.
	SpinnerAnimationMonkey
)

// DoneMsg signals that the spinner operation has completed.
type DoneMsg struct {
	Success bool
	Message string
}

// SpinnerModel represents the spinner component state.
type SpinnerModel struct {
	spinner      spinner.Model
	message      string
	done         bool
	success      bool
	finalMessage string
	style        lipgloss.Style
	errorStyle   lipgloss.Style
	successStyle lipgloss.Style
}

// SpinnerOption is a function that configures a SpinnerModel.
type SpinnerOption func(*SpinnerModel)

// WithSpinnerAnimation sets the spinner animation style.
func WithSpinnerAnimation(anim SpinnerAnimation) SpinnerOption {
	return func(m *SpinnerModel) {
		switch anim {
		case SpinnerAnimationLine:
			m.spinner.Spinner = spinner.Line
		case SpinnerAnimationMiniDot:
			m.spinner.Spinner = spinner.MiniDot
		case SpinnerAnimationJump:
			m.spinner.Spinner = spinner.Jump
		case SpinnerAnimationPulse:
			m.spinner.Spinner = spinner.Pulse
		case SpinnerAnimationPoints:
			m.spinner.Spinner = spinner.Points
		case SpinnerAnimationGlobe:
			m.spinner.Spinner = spinner.Globe
		case SpinnerAnimationMoon:
			m.spinner.Spinner = spinner.Moon
		case SpinnerAnimationMonkey:
			m.spinner.Spinner = spinner.Monkey
		default:
			m.spinner.Spinner = spinner.Dot
		}
	}
}

// WithColor sets the spinner color using a lipgloss color string.
func WithColor(color string) SpinnerOption {
	return func(m *SpinnerModel) {
		m.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(color))
	}
}

// WithMessageStyle sets the style for the spinner message.
func WithMessageStyle(style lipgloss.Style) SpinnerOption {
	return func(m *SpinnerModel) {
		m.style = style
	}
}

// NewSpinnerModel creates a new spinner model with the given message and options.
func NewSpinnerModel(message string, opts ...SpinnerOption) SpinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = SpinnerStyle

	m := SpinnerModel{
		spinner:      s,
		message:      message,
		done:         false,
		success:      false,
		finalMessage: "",
		style:        lipgloss.NewStyle(),
		errorStyle:   ErrorStyle,
		successStyle: SuccessStyle,
	}

	// Apply options
	for _, opt := range opts {
		opt(&m)
	}

	return m
}

// Init initializes the spinner and starts the animation.
func (m SpinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update handles messages and updates the spinner state.
func (m SpinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.done = true
			m.success = false
			m.finalMessage = "Cancelled"
			return m, tea.Quit
		}

	case DoneMsg:
		m.done = true
		m.success = msg.Success
		m.finalMessage = msg.Message
		return m, tea.Quit

	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

// View renders the spinner to a string.
func (m SpinnerModel) View() string {
	if m.done {
		if m.finalMessage != "" {
			if m.success {
				return m.successStyle.Render("✓") + " " + m.finalMessage + "\n"
			}
			return m.errorStyle.Render("✗") + " " + m.finalMessage + "\n"
		}
		return ""
	}

	return m.spinner.View() + " " + m.style.Render(m.message)
}

// Stop marks the spinner as complete.
func (m *SpinnerModel) Stop() {
	m.done = true
}

// SetMessage updates the spinner message.
func (m *SpinnerModel) SetMessage(message string) {
	m.message = message
}

// IsDone returns whether the spinner has finished.
func (m SpinnerModel) IsDone() bool {
	return m.done
}

// IsSuccess returns whether the spinner completed successfully.
func (m SpinnerModel) IsSuccess() bool {
	return m.success
}

// ShowSpinner creates and returns a new tea.Program with the spinner.
// The caller should run this in a goroutine and send DoneMsg when complete.
//
// Example usage:
//
//	p := tui.ShowSpinner("Deploying service...")
//	go func() {
//	    // Perform long-running operation
//	    err := deployService()
//	    if err != nil {
//	        p.Send(tui.DoneMsg{Success: false, Message: err.Error()})
//	    } else {
//	        p.Send(tui.DoneMsg{Success: true, Message: "Deployment complete!"})
//	    }
//	}()
//	if _, err := p.Run(); err != nil {
//	    log.Fatal(err)
//	}
func ShowSpinner(message string, opts ...SpinnerOption) *tea.Program {
	model := NewSpinnerModel(message, opts...)
	return tea.NewProgram(model)
}

// RunSpinnerWithTask executes a task while showing a spinner.
// It handles the spinner lifecycle automatically.
func RunSpinnerWithTask(message string, task func() error, opts ...SpinnerOption) error {
	p := ShowSpinner(message, opts...)

	var taskErr error

	go func() {
		// Small delay to ensure spinner is visible
		time.Sleep(50 * time.Millisecond)

		taskErr = task()

		if taskErr != nil {
			p.Send(DoneMsg{Success: false, Message: fmt.Sprintf("Failed: %v", taskErr)})
		} else {
			p.Send(DoneMsg{Success: true, Message: "Complete"})
		}
	}()

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("spinner error: %w", err)
	}

	return taskErr
}

// RunSpinnerWithTaskAndMessage executes a task while showing a spinner,
// allowing custom success/error messages.
func RunSpinnerWithTaskAndMessage(message string, task func() (string, error), opts ...SpinnerOption) error {
	p := ShowSpinner(message, opts...)

	var taskErr error

	go func() {
		// Small delay to ensure spinner is visible
		time.Sleep(50 * time.Millisecond)

		resultMsg, err := task()
		taskErr = err

		if taskErr != nil {
			errMsg := resultMsg
			if errMsg == "" {
				errMsg = taskErr.Error()
			}
			p.Send(DoneMsg{Success: false, Message: errMsg})
		} else {
			successMsg := resultMsg
			if successMsg == "" {
				successMsg = "Complete"
			}
			p.Send(DoneMsg{Success: true, Message: successMsg})
		}
	}()

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("spinner error: %w", err)
	}

	return taskErr
}

// MultiSpinnerItem represents an item in a multi-spinner display.
type MultiSpinnerItem struct {
	Message  string
	Status   TaskState
	spinner  spinner.Model
	Error    error
	Duration time.Duration
}

// TaskState represents the current status of a spinner item.
type TaskState int

const (
	// TaskStatePending indicates the task has not started.
	TaskStatePending TaskState = iota
	// TaskStateRunning indicates the task is in progress.
	TaskStateRunning
	// TaskStateSuccess indicates the task completed successfully.
	TaskStateSuccess
	// TaskStateFailed indicates the task failed.
	TaskStateFailed
	// TaskStateSkipped indicates the task was skipped.
	TaskStateSkipped
)

// MultiSpinnerModel supports multiple concurrent spinners for parallel tasks.
type MultiSpinnerModel struct {
	items        []MultiSpinnerItem
	currentIndex int
	done         bool
	startTime    time.Time
}

// NewMultiSpinnerModel creates a new multi-spinner model.
func NewMultiSpinnerModel(messages []string) MultiSpinnerModel {
	items := make([]MultiSpinnerItem, len(messages))
	for i, msg := range messages {
		s := spinner.New()
		s.Spinner = spinner.Dot
		s.Style = SpinnerStyle
		items[i] = MultiSpinnerItem{
			Message: msg,
			Status:  TaskStatePending,
			spinner: s,
		}
	}

	return MultiSpinnerModel{
		items:        items,
		currentIndex: 0,
		done:         false,
		startTime:    time.Now(),
	}
}

// Init initializes all spinners.
func (m MultiSpinnerModel) Init() tea.Cmd {
	var cmds []tea.Cmd
	for i := range m.items {
		cmds = append(cmds, m.items[i].spinner.Tick)
	}
	return tea.Batch(cmds...)
}

// UpdateItemMsg updates the status of a specific item.
type UpdateItemMsg struct {
	Index   int
	Status  TaskState
	Error   error
	Message string
}

// Update handles messages for the multi-spinner model.
func (m MultiSpinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.done = true
			return m, tea.Quit
		}

	case UpdateItemMsg:
		if msg.Index >= 0 && msg.Index < len(m.items) {
			m.items[msg.Index].Status = msg.Status
			m.items[msg.Index].Error = msg.Error
			m.items[msg.Index].Duration = time.Since(m.startTime)
			if msg.Message != "" {
				m.items[msg.Index].Message = msg.Message
			}
		}

		// Check if all items are complete
		allDone := true
		for _, item := range m.items {
			if item.Status == TaskStatePending || item.Status == TaskStateRunning {
				allDone = false
				break
			}
		}
		if allDone {
			m.done = true
			return m, tea.Quit
		}

	default:
		// Update all active spinners
		var cmds []tea.Cmd
		for i := range m.items {
			if m.items[i].Status == TaskStateRunning {
				var cmd tea.Cmd
				m.items[i].spinner, cmd = m.items[i].spinner.Update(msg)
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Batch(cmds...)
	}

	return m, nil
}

// View renders the multi-spinner to a string.
func (m MultiSpinnerModel) View() string {
	pendingStyle := MutedStyle
	successStyle := SuccessStyle
	failedStyle := ErrorStyle
	skippedStyle := WarningStyle

	var s string
	for _, item := range m.items {
		var prefix string
		var msgStyle lipgloss.Style

		switch item.Status {
		case TaskStatePending:
			prefix = pendingStyle.Render("○")
			msgStyle = pendingStyle
		case TaskStateRunning:
			prefix = item.spinner.View()
			msgStyle = lipgloss.NewStyle()
		case TaskStateSuccess:
			prefix = successStyle.Render("✓")
			msgStyle = successStyle
		case TaskStateFailed:
			prefix = failedStyle.Render("✗")
			msgStyle = failedStyle
		case TaskStateSkipped:
			prefix = skippedStyle.Render("-")
			msgStyle = skippedStyle
		}

		s += prefix + " " + msgStyle.Render(item.Message)
		if item.Error != nil {
			s += failedStyle.Render(fmt.Sprintf(" (%v)", item.Error))
		}
		s += "\n"
	}

	return s
}

// ShowMultiSpinner creates and returns a tea.Program for multi-spinner display.
func ShowMultiSpinner(messages []string) *tea.Program {
	model := NewMultiSpinnerModel(messages)
	return tea.NewProgram(model)
}
