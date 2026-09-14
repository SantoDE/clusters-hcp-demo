package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func statusLabel(s clusterState, noWorkers bool) string {
	type entry struct {
		label string
		color lipgloss.Color
	}
	var e entry
	switch {
	case s.startTime.IsZero():
		e = entry{"Not started", "240"}
	case s.rancherActive.done && (noWorkers || s.workersReady.done):
		e = entry{"Active", "82"}
	case noWorkers && s.cpReady.done:
		e = entry{"CP Ready", "226"}
	case !noWorkers && s.workersReady.done:
		e = entry{"Workers Ready", "226"}
	case s.cpReady.done:
		e = entry{"Workers Joining", "214"}
	default:
		e = entry{"Provisioning", "214"}
	}
	return lipgloss.NewStyle().Bold(true).Foreground(e.color).Render(e.label)
}

func dot(on bool) string {
	if on {
		return okDotSt.Render("●")
	}
	return dimSt.Render("○")
}

func msRow(label string, ms milestone, start time.Time, na bool) string {
	if na {
		return fmt.Sprintf("%s %-16s %s", dimSt.Render("─"), msSt.Render(label), naSt.Render("N/A (shared)"))
	}
	d := dot(ms.done)
	suffix := ""
	if ms.done && ms.at != nil && !start.IsZero() {
		elapsed := ms.at.Sub(start).Round(time.Second)
		suffix = msTimeSt.Render(fmt.Sprintf("+%s", elapsed))
	}
	return fmt.Sprintf("%s %-16s %s", d, msSt.Render(label), suffix)
}

func memBar(used, maxVal int64, width int) string {
	if maxVal == 0 {
		return dimSt.Render(strings.Repeat("░", width))
	}
	fill := int(float64(used) / float64(maxVal) * float64(width))
	if fill > width {
		fill = width
	}
	bar := lipgloss.NewStyle().Foreground(lipgloss.Color("75")).Render(strings.Repeat("█", fill))
	bar += dimSt.Render(strings.Repeat("░", width-fill))
	return bar
}

func fmtMem(mi int64) string {
	if mi == 0 {
		return "—"
	}
	if mi >= 1024 {
		return fmt.Sprintf("%.1fGi", float64(mi)/1024)
	}
	return fmt.Sprintf("%dMi", mi)
}

func fmtCPU(milli int64) string {
	if milli == 0 {
		return "—"
	}
	if milli >= 1000 {
		return fmt.Sprintf("%.2f", float64(milli)/1000)
	}
	return fmt.Sprintf("%dm", milli)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func maxInt64(vals ...int64) int64 {
	var m int64
	for _, v := range vals {
		if v > m {
			m = v
		}
	}
	return m
}

// ─── view ─────────────────────────────────────────────────────────────────────

func (m model) View() string {
	panels := make([]string, 3)

	maxMem := maxInt64(512, m.states[0].totalMem, m.states[1].totalMem, m.states[2].totalMem)
	maxCPU := maxInt64(100, m.states[0].totalCPU, m.states[1].totalCPU, m.states[2].totalCPU)

	for i, def := range defs {
		s := m.states[i]
		on := m.selected[i]

		if m.deleting[i] {
			numKey := dimSt.Render(fmt.Sprintf("[%d]", i+1))
			titleLine := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("238")).
				Background(lipgloss.Color("236")).Padding(0, 1).Width(panelW - 4).Render(def.label)
			content := titleLine + "\n" +
				dimSt.Render(def.subtitle) + "\n\n" +
				lipgloss.NewStyle().Foreground(lipgloss.Color("160")).Bold(true).Render("Deleting…") + "  " + numKey + "\n"
			panels[i] = panelDeletingSt.Render(content)
			continue
		}

		if !on && !m.states[i].startTime.IsZero() {
			numKey := dimSt.Render(fmt.Sprintf("[%d]", i+1))
			titleLine := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("238")).
				Background(lipgloss.Color("236")).Padding(0, 1).Width(panelW - 4).Render(def.label)
			content := titleLine + "\n" +
				dimSt.Render(def.subtitle) + "\n\n" +
				lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true).Render("⚠ pending delete") + "  " + numKey + "\n"
			panels[i] = panelPendingDeleteSt.Render(content)
			continue
		}

		if !on {
			numKey := dimSt.Render(fmt.Sprintf("[%d]", i+1))
			titleLine := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("238")).
				Background(lipgloss.Color("236")).Padding(0, 1).Width(panelW - 4).Render(def.label)
			content := titleLine + "\n" +
				dimSt.Render(def.subtitle) + "\n\n" +
				dimSt.Render("skipped") + "  " + numKey + "\n"
			panels[i] = panelOffSt.Render(content)
			continue
		}

		// overall timer
		noWorkers := def.mdName == ""
		allDone := s.rancherActive.done && (noWorkers || s.workersReady.done)
		var doneAt *time.Time
		if allDone {
			doneAt = s.rancherActive.at
			if !noWorkers && s.workersReady.at != nil {
				if doneAt == nil || s.workersReady.at.After(*doneAt) {
					doneAt = s.workersReady.at
				}
			}
		}
		var timerStr string
		switch {
		case s.startTime.IsZero():
			timerStr = dimSt.Render("not started")
		case allDone && doneAt != nil:
			elapsed := doneAt.Sub(s.startTime).Round(time.Second)
			timerStr = doneTimeSt.Render(fmt.Sprintf("✓ %s", elapsed))
		default:
			timerStr = timerSt.Render(fmt.Sprintf("⏱ %s", time.Since(s.startTime).Round(time.Second)))
		}

		// milestones
		ms1 := msRow("CP Ready", s.cpReady, s.startTime, false)
		ms2 := msRow("Workers Ready", s.workersReady, s.startTime, noWorkers)
		ms3 := msRow("Rancher Active", s.rancherActive, s.startTime, false)

		// pod list (max 5)
		podLines := ""
		pods := s.pods
		if len(pods) > 5 {
			pods = pods[:5]
		}
		for _, p := range pods {
			podLines += fmt.Sprintf("%s  %s\n",
				podNameSt.Render(truncate(p.name, panelW-14)),
				podStatSt.Render(p.status))
		}
		if len(s.pods) > 5 {
			podLines += dimSt.Render(fmt.Sprintf("… +%d more", len(s.pods)-5)) + "\n"
		}

		numKey := dimSt.Render(fmt.Sprintf("[%d]", i+1))
		content := titleSt.Render(def.label) + " " + numKey + "\n" +
			subtitleSt.Render(def.subtitle) + "\n\n" +
			statusLabel(s, noWorkers) + "  " + timerStr + "\n\n" +
			ms1 + "\n" +
			ms2 + "\n" +
			ms3 + "\n"

		if podLines != "" {
			content += "\n" + podLines
		}

		st := panelSt
		if allDone {
			st = panelDoneSt
		}
		panels[i] = st.Render(content)
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, panels[0], "  ", panels[1], "  ", panels[2])

	nSel := 0
	for _, on := range m.selected {
		if on {
			nSel++
		}
	}

	var help string
	switch m.phase {
	case phaseIdle:
		help = helpSt.Render(fmt.Sprintf("  [1/2/3] toggle   [s] start %d   [q] quit", nSel))
	case phaseConfirm:
		nAdd, nDel := 0, 0
		for i := range m.states {
			if m.selected[i] && m.states[i].startTime.IsZero() {
				nAdd++
			}
			if !m.selected[i] && !m.states[i].startTime.IsZero() && !m.deleting[i] {
				nDel++
			}
		}
		var parts []string
		if nAdd > 0 {
			parts = append(parts, fmt.Sprintf("apply %d", nAdd))
		}
		if nDel > 0 {
			parts = append(parts, fmt.Sprintf("delete %d", nDel))
		}
		help = warnSt.Render(fmt.Sprintf("  %s? [s] confirm   [esc] cancel", strings.Join(parts, ", ")))
	case phaseRunning:
		if m.hasPendingAction() {
			help = helpSt.Render("  [1/2/3] toggle   [s] confirm   [q] quit")
		} else {
			help = helpSt.Render("  [1/2/3] toggle   [q] quit")
		}
	}

	resources := resourceSection(m.states, maxCPU, maxMem)

	return "\n" + row + "\n\n" + resources + "\n" + help + "\n"
}

func resourceSection(states [3]clusterState, maxCPU, maxMem int64) string {
	const barW = 24
	const labelW = 18

	heading := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252"))

	row := func(label string, used, max int64, fmtVal func(int64) string) string {
		return fmt.Sprintf("  %-*s  %s  %s\n",
			labelW, label,
			memBar(used, max, barW),
			fmtVal(used))
	}

	cpu := heading.Render("CPU") + "\n"
	for i, s := range states {
		cpu += row(defs[i].label, s.totalCPU, maxCPU, fmtCPU)
	}

	mem := heading.Render("Memory") + "\n"
	for i, s := range states {
		mem += row(defs[i].label, s.totalMem, maxMem, fmtMem)
	}

	return cpu + "\n" + mem
}
