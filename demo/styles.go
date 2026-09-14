package main

import "github.com/charmbracelet/lipgloss"

var (
	panelW = 38

	titleSt = lipgloss.NewStyle().Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("62")).
		Padding(0, 1).Width(panelW - 4)

	subtitleSt = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Italic(true)

	panelSt = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).Width(panelW)

	panelDoneSt = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("82")).
		Padding(1, 2).Width(panelW)

	panelOffSt = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("236")).
		Padding(1, 2).Width(panelW)

	panelPendingDeleteSt = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("208")).
		Padding(1, 2).Width(panelW)

	panelDeletingSt = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("160")).
		Padding(1, 2).Width(panelW)

	timerSt    = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	doneTimeSt = lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
	dimSt      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	okDotSt    = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	msSt       = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	msTimeSt   = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	naSt       = lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Italic(true)
	podNameSt  = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	podStatSt  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	helpSt     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	warnSt     = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
)
