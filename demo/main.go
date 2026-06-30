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
	podNS      string
	cpGVR      schema.GroupVersionResource
	cpName     string
	mdName     string // empty = no dedicated workers (k3k shared mode)
	applyFiles []string
}

var defs = []clusterDef{
	{
		label:    "KubeVirt k3s",
		subtitle: "VMs all the way down",
		capiNS:   "capi-k3s-kubevirt",
		capiName: "k3s-kubevirt",
		podNS:    "capi-k3s-kubevirt",
		cpGVR:    schema.GroupVersionResource{Group: "controlplane.cluster.x-k8s.io", Version: "v1beta1", Resource: "kthreescontrolplanes"},
		cpName:   "k3s-kubevirt-cp",
		mdName:   "k3s-kubevirt-md-0",
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
		cpGVR:    schema.GroupVersionResource{Group: "controlplane.cluster.x-k8s.io", Version: "v1alpha2", Resource: "kamajicontrolplanes"},
		cpName:   "kamaji-kubevirt-cp",
		mdName:   "kamaji-kubevirt-md-0",
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
		cpGVR:    schema.GroupVersionResource{Group: "controlplane.cluster.x-k8s.io", Version: "v1beta1", Resource: "k3kcontrolplanes"},
		cpName:   "k3k-simple",
		mdName:   "", // shared mode — no worker VMs
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
	mdGVR = schema.GroupVersionResource{
		Group: "cluster.x-k8s.io", Version: "v1beta1", Resource: "machinedeployments",
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

type milestone struct {
	done bool
	at   *time.Time // wall time when first reached
}

func (ms *milestone) reach(prev milestone) {
	if !prev.done && ms.done && ms.at == nil {
		t := time.Now()
		ms.at = &t
	} else if prev.at != nil {
		ms.at = prev.at
	}
}

type podInfo struct {
	name   string
	status string
}

type clusterState struct {
	phase    string
	pods     []podInfo
	totalCPU int64
	totalMem int64
	startTime time.Time

	cpReady      milestone
	workersReady milestone
	rancherActive milestone
}

type appPhase int

const (
	phaseIdle appPhase = iota
	phaseConfirm
	phaseRunning
)

type model struct {
	client dynamic.Interface
	phase  appPhase
	states [3]clusterState
	width  int
	height int
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
	return model{client: client}
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
			startTime:    prev.startTime,
			cpReady:      milestone{done: prev.cpReady.done, at: prev.cpReady.at},
			workersReady: milestone{done: prev.workersReady.done, at: prev.workersReady.at},
			rancherActive: milestone{done: prev.rancherActive.done, at: prev.rancherActive.at},
		}

		// CAPI cluster phase
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

		// CP ready — check status.ready or status.initialized on the CP object
		cpObj, err := client.Resource(def.cpGVR).Namespace(def.capiNS).Get(ctx, def.cpName, metav1.GetOptions{})
		if err == nil {
			cpStatus, _ := cpObj.Object["status"].(map[string]interface{})
			ready, _ := cpStatus["ready"].(bool)
			initialized, _ := cpStatus["initialized"].(bool)
			next.cpReady.done = ready || initialized
		}
		next.cpReady.reach(prev.cpReady)

		// Workers ready — MachineDeployment readyReplicas >= 1
		if def.mdName != "" {
			mdObj, err := client.Resource(mdGVR).Namespace(def.capiNS).Get(ctx, def.mdName, metav1.GetOptions{})
			if err == nil {
				mdStatus, _ := mdObj.Object["status"].(map[string]interface{})
				ready, _ := mdStatus["readyReplicas"].(int64)
				next.workersReady.done = ready >= 1
			}
		}
		next.workersReady.reach(prev.workersReady)

		// Rancher Active — match provisioning cluster by Turtles owner annotations
		pList, _ := client.Resource(provisioningGVR).Namespace("fleet-default").List(ctx, metav1.ListOptions{})
		if pList != nil {
			for _, item := range pList.Items {
				meta, _ := item.Object["metadata"].(map[string]interface{})
				ann, _ := meta["annotations"].(map[string]interface{})
				if ann["cluster-api.cattle.io/capi-cluster-owner-namespace"] == def.capiNS &&
					ann["cluster-api.cattle.io/capi-cluster-owner-name"] == def.capiName {
					st, _ := item.Object["status"].(map[string]interface{})
					next.rancherActive.done, _ = st["ready"].(bool)
					break
				}
			}
		}
		next.rancherActive.reach(prev.rancherActive)

		// pods + metrics
		pods, _ := client.Resource(podGVR).Namespace(def.podNS).List(ctx, metav1.ListOptions{})
		metrics, _ := client.Resource(metricsGVR).Namespace(def.podNS).List(ctx, metav1.ListOptions{})

		metricsByPod := map[string][2]int64{}
		if metrics != nil {
			for _, pm := range metrics.Items {
				meta, _ := pm.Object["metadata"].(map[string]interface{})
				podName, _ := meta["name"].(string)
				var cpuM, memMi int64
				for _, c := range pm.Object["containers"].([]interface{}) {
					cm, _ := c.(map[string]interface{})
					usage, _ := cm["usage"].(map[string]interface{})
					if s, ok := usage["cpu"].(string); ok {
						if q, err := resource.ParseQuantity(s); err == nil {
							cpuM += q.MilliValue()
						}
					}
					if s, ok := usage["memory"].(string); ok {
						if q, err := resource.ParseQuantity(s); err == nil {
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
				next.pods = append(next.pods, podInfo{name: name, status: phase})
				next.totalCPU += m[0]
				next.totalMem += m[1]
			}
		}

		return pollResultMsg{idx: idx, state: next}
	}
}

func applyAllCmd() tea.Cmd {
	return func() tea.Msg {
		for _, def := range defs {
			for _, f := range def.applyFiles {
				exec.Command("kubectl", "--context", "ranchero-k3s", "apply", "-f", f).Run()
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
		// nothing; poll loop picks up state changes

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
	okDotSt    = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	msSt       = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	msTimeSt   = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	naSt       = lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Italic(true)
	podNameSt  = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
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

		// overall timer
		var timerStr string
		switch {
		case s.startTime.IsZero():
			timerStr = dimSt.Render("not started")
		case s.rancherActive.done && s.rancherActive.at != nil:
			elapsed := s.rancherActive.at.Sub(s.startTime).Round(time.Second)
			timerStr = doneTimeSt.Render(fmt.Sprintf("✓ %s", elapsed))
		default:
			timerStr = timerSt.Render(fmt.Sprintf("⏱ %s", time.Since(s.startTime).Round(time.Second)))
		}

		// milestones
		noWorkers := def.mdName == ""
		ms1 := msRow("CP Ready", s.cpReady, s.startTime, false)
		ms2 := msRow("Workers Ready", s.workersReady, s.startTime, noWorkers)
		ms3 := msRow("Rancher Active", s.rancherActive, s.startTime, false)

		// resource bars
		barW := 10
		cpuLine := fmt.Sprintf("CPU  %s  %s", memBar(s.totalCPU, maxCPU, barW), fmtCPU(s.totalCPU))
		memLine := fmt.Sprintf("MEM  %s  %s", memBar(s.totalMem, maxMem, barW), fmtMem(s.totalMem))

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

		content := titleSt.Render(def.label) + "\n" +
			subtitleSt.Render(def.subtitle) + "\n\n" +
			phaseSt(s.phase) + "  " + timerStr + "\n\n" +
			ms1 + "\n" +
			ms2 + "\n" +
			ms3 + "\n\n" +
			cpuLine + "\n" +
			memLine + "\n"

		if podLines != "" {
			content += "\n" + podLines
		}

		st := panelSt
		if s.rancherActive.done {
			st = panelDoneSt
		}
		panels[i] = st.Render(content)
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, panels[0], "  ", panels[1], "  ", panels[2])

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

// ─── main ─────────────────────────────────────────────────────────────────────

func main() {
	p := tea.NewProgram(newModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
