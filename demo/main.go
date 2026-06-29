package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

// ─── cluster definitions ──────────────────────────────────────────────────────

type clusterDef struct {
	label      string
	subtitle   string
	capiNS     string
	capiName   string
	podNS      string // namespace where the interesting pods live
	applyFiles []string
}

var defs = []clusterDef{
	{
		label:    "KubeVirt k3s",
		subtitle: "VMs all the way down",
		capiNS:   "capi-k3s-kubevirt",
		capiName: "k3s-kubevirt",
		podNS:    "capi-k3s-kubevirt",
		applyFiles: []string{
			"clusters/k3s-kubevirt/cluster.yaml",
		},
	},
	{
		label:    "Kamaji + KubeVirt",
		subtitle: "Hosted CP, VM workers",
		capiNS:   "capi-kamaji-kubevirt",
		capiName: "kamaji-kubevirt",
		podNS:    "capi-kamaji-kubevirt",
		applyFiles: []string{
			"clusters/kamaji-kubevirt/cluster.yaml",
			"clusters/kamaji-kubevirt/cni-configmap.yaml",
			"clusters/kamaji-kubevirt/cni.yaml",
		},
	},
	{
		label:    "k3k",
		subtitle: "Pure pods, no VMs",
		capiNS:   "capi-k3k",
		capiName: "k3k-simple",
		podNS:    "k3k-k3k-simple",
		applyFiles: []string{
			"clusters/k3k/provider.yaml",
			"clusters/k3k/cluster.yaml",
		},
	},
}

// ─── GVRs ────────────────────────────────────────────────────────────────────

var (
	clusterGVR = schema.GroupVersionResource{
		Group: "cluster.x-k8s.io", Version: "v1beta1", Resource: "clusters",
	}
	provisioningGVR = schema.GroupVersionResource{
		Group: "provisioning.cattle.io", Version: "v1", Resource: "clusters",
	}
	podGVR = schema.GroupVersionResource{
		Version: "v1", Resource: "pods",
	}
	metricsGVR = schema.GroupVersionResource{
		Group: "metrics.k8s.io", Version: "v1beta1", Resource: "pods",
	}
)

// ─── state ────────────────────────────────────────────────────────────────────

type podInfo struct {
	name   string
	status string
	cpuM   int64
	memMi  int64
}

type clusterState struct {
	phase     string
	available bool
	rancher   bool
	events    []string
	startTime time.Time
	doneAt    *time.Time
	pods      []podInfo
	totalCPU  int64
	totalMem  int64
}

type appPhase int

const (
	phaseIdle appPhase = iota
	phaseConfirm
	phaseRunning
)

type model struct {
	client   dynamic.Interface
	phase    appPhase
	states   [3]clusterState
	width    int
	height   int
}

// ─── messages ─────────────────────────────────────────────────────────────────

type tickMsg struct{}
type pollResultMsg struct {
	idx   int
	state clusterState
}
type applyDoneMsg struct{}

// ─── init ─────────────────────────────────────────────────────────────────────

func newModel() model {
	kc := os.Getenv("KUBECONFIG")
	if kc == "" {
		home, _ := os.UserHomeDir()
		kc = home + "/.kube/config"
	}
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kc},
		&clientcmd.ConfigOverrides{CurrentContext: "ranchero-k3s"},
	).ClientConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "kubeconfig: %v\n", err)
		os.Exit(1)
	}
	client, err := dynamic.NewForConfig(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "client: %v\n", err)
		os.Exit(1)
	}
	var states [3]clusterState
	for i := range states {
		states[i] = clusterState{events: []string{}}
	}
	return model{client: client, states: states}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), m.pollAllCmd())
}

// ─── commands ─────────────────────────────────────────────────────────────────

func tickCmd() tea.Cmd {
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg { return tickMsg{} })
}

func (m model) pollAllCmd() tea.Cmd {
	cmds := make([]tea.Cmd, 3)
	for i, def := range defs {
		cmds[i] = m.pollClusterCmd(i, def)
	}
	return tea.Batch(cmds...)
}

func (m model) pollClusterCmd(idx int, def clusterDef) tea.Cmd {
	client := m.client
	prev := m.states[idx]
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		defer cancel()

		next := clusterState{
			events:    prev.events,
			startTime: prev.startTime,
			doneAt:    prev.doneAt,
		}

		// CAPI cluster
		obj, err := client.Resource(clusterGVR).Namespace(def.capiNS).Get(ctx, def.capiName, metav1.GetOptions{})
		if err != nil {
			next.phase = "Not found"
			return pollResultMsg{idx: idx, state: next}
		}
		status, _ := obj.Object["status"].(map[string]interface{})
		next.phase, _ = status["phase"].(string)
		if next.phase == "" {
			next.phase = "Pending"
		}
		if conds, ok := status["conditions"].([]interface{}); ok {
			for _, c := range conds {
				cm, _ := c.(map[string]interface{})
				if cm["type"] == "Available" && cm["status"] == "True" {
					next.available = true
				}
			}
		}

		// Rancher provisioning cluster — match by Turtles owner annotations
		pList, err := client.Resource(provisioningGVR).Namespace("fleet-default").List(ctx, metav1.ListOptions{})
		if err == nil {
			for _, item := range pList.Items {
				ann, _ := item.Object["metadata"].(map[string]interface{})
				annMap, _ := ann["annotations"].(map[string]interface{})
				ownerNS, _ := annMap["cluster-api.cattle.io/capi-cluster-owner-namespace"].(string)
				ownerName, _ := annMap["cluster-api.cattle.io/capi-cluster-owner-name"].(string)
				if ownerNS == def.capiNS && ownerName == def.capiName {
					st, _ := item.Object["status"].(map[string]interface{})
					next.rancher, _ = st["ready"].(bool)
					break
				}
			}
		}

		// pods in podNS
		pods, _ := client.Resource(podGVR).Namespace(def.podNS).List(ctx, metav1.ListOptions{})
		metrics, _ := client.Resource(metricsGVR).Namespace(def.podNS).List(ctx, metav1.ListOptions{})

		metricsByPod := map[string][2]int64{} // name → [cpuMilli, memMi]
		if metrics != nil {
			for _, m := range metrics.Items {
				meta, _ := m.Object["metadata"].(map[string]interface{})
				podName, _ := meta["name"].(string)
				containers, _ := m.Object["containers"].([]interface{})
				var cpuM, memMi int64
				for _, c := range containers {
					cm, _ := c.(map[string]interface{})
					usage, _ := cm["usage"].(map[string]interface{})
					if cpuStr, ok := usage["cpu"].(string); ok {
						q, err := resource.ParseQuantity(cpuStr)
						if err == nil {
							cpuM += q.MilliValue()
						}
					}
					if memStr, ok := usage["memory"].(string); ok {
						q, err := resource.ParseQuantity(memStr)
						if err == nil {
							memMi += q.Value() / (1024 * 1024)
						}
					}
				}
				metricsByPod[podName] = [2]int64{cpuM, memMi}
			}
		}

		if pods != nil {
			for _, pod := range pods.Items {
				meta, _ := pod.Object["metadata"].(map[string]interface{})
				name, _ := meta["name"].(string)
				podStatus, _ := pod.Object["status"].(map[string]interface{})
				phase, _ := podStatus["phase"].(string)
				m := metricsByPod[name]
				next.pods = append(next.pods, podInfo{
					name:   name,
					status: phase,
					cpuM:   m[0],
					memMi:  m[1],
				})
				next.totalCPU += m[0]
				next.totalMem += m[1]
			}
		}

		// event log on phase change
		if prev.phase != next.phase && next.phase != "" && prev.phase != "" {
			entry := fmt.Sprintf("%s → %s", prev.phase, next.phase)
			next.events = append(next.events, entry)
			if len(next.events) > 5 {
				next.events = next.events[len(next.events)-5:]
			}
		}

		// capture done time
		if !prev.rancher && next.rancher {
			t := time.Now()
			next.doneAt = &t
		}

		return pollResultMsg{idx: idx, state: next}
	}
}

func applyAllCmd() tea.Cmd {
	return func() tea.Msg {
		for _, def := range defs {
			for _, f := range def.applyFiles {
				cmd := exec.Command("kubectl", "--context", "ranchero-k3s", "apply", "-f", f)
				cmd.Run()
			}
		}
		return applyDoneMsg{}
	}
}

// ─── update ───────────────────────────────────────────────────────────────────

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "s":
			switch m.phase {
			case phaseIdle:
				m.phase = phaseConfirm
			case phaseConfirm:
				m.phase = phaseRunning
				now := time.Now()
				for i := range m.states {
					m.states[i].startTime = now
					m.states[i].phase = "Pending"
				}
				return m, applyAllCmd()
			}
		case "esc":
			if m.phase == phaseConfirm {
				m.phase = phaseIdle
			}
		}

	case tickMsg:
		return m, tea.Batch(tickCmd(), m.pollAllCmd())

	case applyDoneMsg:
		return m, nil

	case pollResultMsg:
		m.states[msg.idx] = msg.state
	}

	return m, nil
}

// ─── styles ───────────────────────────────────────────────────────────────────

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

	timerSt    = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	doneTimeSt = lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
	dimSt      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	eventSt    = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	okDotSt    = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	podNameSt  = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	podStatSt  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	helpSt     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	warnSt     = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)

	phaseColors = map[string]lipgloss.Color{
		"Provisioning": "214",
		"Provisioned":  "226",
		"Not found":    "240",
		"Pending":      "240",
		"Deleting":     "196",
	}
)

func phaseSt(phase string) string {
	col, ok := phaseColors[phase]
	if !ok {
		col = "250"
	}
	return lipgloss.NewStyle().Bold(true).Foreground(col).Render(phase)
}

func dot(on bool) string {
	if on {
		return okDotSt.Render("●")
	}
	return dimSt.Render("○")
}

func memBar(used, maxMi int64, width int) string {
	if maxMi == 0 {
		return dimSt.Render(strings.Repeat("░", width))
	}
	fill := int(float64(used) / float64(maxMi) * float64(width))
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

// ─── view ─────────────────────────────────────────────────────────────────────

func (m model) View() string {
	panels := make([]string, 3)

	// find max mem across clusters for relative bar scale
	var maxMem int64 = 512 // floor so bar isn't empty when mem is low
	for _, s := range m.states {
		if s.totalMem > maxMem {
			maxMem = s.totalMem
		}
	}

	for i, def := range defs {
		s := m.states[i]

		// timer
		var timerStr string
		if s.startTime.IsZero() {
			timerStr = dimSt.Render("not started")
		} else if s.rancher && s.doneAt != nil {
			elapsed := s.doneAt.Sub(s.startTime).Round(time.Second)
			timerStr = doneTimeSt.Render(fmt.Sprintf("✓ %s", elapsed))
		} else if !s.startTime.IsZero() {
			elapsed := time.Since(s.startTime).Round(time.Second)
			timerStr = timerSt.Render(fmt.Sprintf("⏱ %s", elapsed))
		}

		// phase line
		phaseLine := fmt.Sprintf("%s  %s", phaseSt(s.phase), timerStr)

		// status dots
		dots := fmt.Sprintf("%s CAPI Available\n%s Rancher Active",
			dot(s.available), dot(s.rancher))

		// resource bars
		barW := 10
		memLine := fmt.Sprintf("MEM  %s  %s", memBar(s.totalMem, maxMem, barW), fmtMem(s.totalMem))
		cpuLine := fmt.Sprintf("CPU  %s  %s", memBar(s.totalCPU, max64(maxCPU(m), 100), barW), fmtCPU(s.totalCPU))

		// pod list (max 6)
		podLines := ""
		pods := s.pods
		if len(pods) > 6 {
			pods = pods[:6]
		}
		for _, p := range pods {
			name := truncate(p.name, panelW-16)
			stat := podStatSt.Render(p.status)
			podLines += fmt.Sprintf("%s  %s\n", podNameSt.Render(name), stat)
		}
		if len(s.pods) > 6 {
			podLines += dimSt.Render(fmt.Sprintf("  … +%d more", len(s.pods)-6)) + "\n"
		}

		// event log
		evLines := ""
		for _, e := range s.events {
			evLines += eventSt.Render("  "+e) + "\n"
		}

		content := titleSt.Render(def.label) + "\n" +
			subtitleSt.Render(def.subtitle) + "\n\n" +
			phaseLine + "\n\n" +
			dots + "\n\n" +
			cpuLine + "\n" +
			memLine + "\n"

		if podLines != "" {
			content += "\n" + podLines
		}
		if evLines != "" {
			content += "\n" + evLines
		}

		st := panelSt
		if s.rancher {
			st = panelDoneSt
		}
		panels[i] = st.Render(content)
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, panels[0], "  ", panels[1], "  ", panels[2])

	// help / confirm bar
	var help string
	switch m.phase {
	case phaseIdle:
		help = helpSt.Render("  [s] start all   [q] quit")
	case phaseConfirm:
		help = warnSt.Render("  Apply all 3 clusters? [s] confirm   [esc] cancel")
	case phaseRunning:
		help = helpSt.Render("  [q] quit")
	}

	return "\n" + row + "\n\n" + help + "\n"
}

func maxCPU(m model) int64 {
	var max int64 = 100
	for _, s := range m.states {
		if s.totalCPU > max {
			max = s.totalCPU
		}
	}
	return max
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// ─── main ─────────────────────────────────────────────────────────────────────

func main() {
	p := tea.NewProgram(newModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
