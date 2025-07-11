package cli

// Copyright (C) 2025 Rizome Labs, Inc.
//
// This program is free software; you can redistribute it and/or
// modify it under the terms of the GNU General Public License
// as published by the Free Software Foundation; either version 2
// of the License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program; if not, write to the Free Software
// Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301, USA.

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// progressModel shows a progress bar for long-running operations
type progressModel struct {
	progress progress.Model
	title    string
	percent  float64
	done     bool
}

func initialProgressModel(title string) progressModel {
	return progressModel{
		progress: progress.New(progress.WithDefaultGradient()),
		title:    title,
		percent:  0.0,
		done:     false,
	}
}

type progressMsg float64
type doneMsg struct{}

func (m progressModel) Init() tea.Cmd {
	return nil
}

func (m progressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.progress.Width = msg.Width - 4
		return m, nil

	case progressMsg:
		m.percent = float64(msg)
		if m.percent >= 1.0 {
			m.done = true
			return m, tea.Quit
		}
		return m, nil

	case doneMsg:
		m.done = true
		return m, tea.Quit

	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m progressModel) View() string {
	if m.done {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Render("✓ " + m.title + " completed!")
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205"))

	return fmt.Sprintf(
		"%s\n\n%s %.0f%%",
		titleStyle.Render(m.title),
		m.progress.ViewAs(m.percent),
		m.percent*100,
	)
}

// Progress creates a progress bar that can be updated
type Progress struct {
	program *tea.Program
}

// NewProgress creates a new progress indicator
func NewProgress(title string) *Progress {
	p := tea.NewProgram(initialProgressModel(title))
	go p.Run()

	return &Progress{
		program: p,
	}
}

// Update updates the progress percentage (0.0 to 1.0)
func (p *Progress) Update(percent float64) {
	p.program.Send(progressMsg(percent))
}

// Done marks the progress as complete
func (p *Progress) Done() {
	p.program.Send(doneMsg{})
	time.Sleep(100 * time.Millisecond) // Give it time to display
}

// statusModel shows a spinner with status text
type statusModel struct {
	spinner spinner.Model
	status  string
	done    bool
	err     error
}

func initialStatusModel(status string) statusModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return statusModel{
		spinner: s,
		status:  status,
		done:    false,
	}
}

type statusMsg string
type statusDoneMsg struct{}
type statusErrorMsg error

func (m statusModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m statusModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case statusMsg:
		m.status = string(msg)
		return m, nil

	case statusDoneMsg:
		m.done = true
		return m, tea.Quit

	case statusErrorMsg:
		m.err = msg
		m.done = true
		return m, tea.Quit

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m statusModel) View() string {
	if m.done {
		if m.err != nil {
			return lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Render("✗ " + m.status + " failed: " + m.err.Error())
		}
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Render("✓ " + m.status + " completed!")
	}

	return fmt.Sprintf("%s %s", m.spinner.View(), m.status)
}

// Status creates a spinner with status text
type Status struct {
	program *tea.Program
}

// NewStatus creates a new status indicator
func NewStatus(initialStatus string) *Status {
	p := tea.NewProgram(initialStatusModel(initialStatus))
	go p.Run()

	return &Status{
		program: p,
	}
}

// Update updates the status text
func (s *Status) Update(status string) {
	s.program.Send(statusMsg(status))
}

// Done marks the status as complete
func (s *Status) Done() {
	s.program.Send(statusDoneMsg{})
	time.Sleep(100 * time.Millisecond) // Give it time to display
}

// Error marks the status as failed
func (s *Status) Error(err error) {
	s.program.Send(statusErrorMsg(err))
	time.Sleep(100 * time.Millisecond) // Give it time to display
}
