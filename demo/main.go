package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

// ─── cluster definitions ──────────────────────────────────────────────────────

type clusterDef struct {
	label          string
	subtitle       string
	capiNS         string
	capiName       string
	podNS          string
	cpGVR          schema.GroupVersionResource
	cpName         string
	mdName         string // empty = no dedicated workers (k3k shared mode)
	extraNamespace string // namespace to delete on top of capiNS (e.g. k3k host ns)
	applyFiles     []string
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
		label:          "k3k",
		subtitle:       "Pure pods, no VMs",
		capiNS:         "capi-k3k",
		capiName:       "k3k-simple",
		podNS:          "k3k-k3k-simple",
		cpGVR:          schema.GroupVersionResource{Group: "controlplane.cluster.x-k8s.io", Version: "v1beta1", Resource: "k3kcontrolplanes"},
		cpName:         "k3k-simple",
		mdName:         "", // shared mode — no worker VMs
		extraNamespace: "k3k-k3k-simple",
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
		Group: "cluster.x-k8s.io", Version: "v1beta2", Resource: "machinedeployments",
	}
	provisioningGVR = schema.GroupVersionResource{
		Group: "provisioning.cattle.io", Version: "v1", Resource: "clusters",
	}
	managementClusterGVR = schema.GroupVersionResource{
		Group: "management.cattle.io", Version: "v3", Resource: "clusters",
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
	client   dynamic.Interface
	phase    appPhase
	selected [3]bool
	deleting [3]bool
	states   [3]clusterState
	width    int
	height   int
}

// ─── messages ─────────────────────────────────────────────────────────────────

type tickMsg struct{}
type pollResultMsg struct {
	idx        int
	state      clusterState
	discovered bool // true on the first poll that finds an existing cluster
}
type applyDoneMsg struct{}
type clusterDeletedMsg struct{ idx int }

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
	cfg.WarningHandler = rest.NoWarnings{}
	cfg.QPS = 50
	cfg.Burst = 100
	client, err := dynamic.NewForConfig(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "client: %v\n", err)
		os.Exit(1)
	}
	return model{client: client, selected: [3]bool{true, true, true}}
}

func (m model) hasPendingAction() bool {
	for i := range m.states {
		if m.selected[i] && m.states[i].startTime.IsZero() {
			return true
		}
		if !m.selected[i] && !m.states[i].startTime.IsZero() && !m.deleting[i] {
			return true
		}
	}
	return false
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

// toFloat64 handles the json.Number type returned by the Kubernetes unstructured
// decoder (which uses UseNumber), as well as plain float64 from tests/mocks.
func toFloat64(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int64:
		return float64(n)
	case int32:
		return float64(n)
	case interface{ Float64() (float64, error) }: // json.Number
		f, _ := n.Float64()
		return f
	}
	return 0
}

// conditionTime returns the lastTransitionTime of the first condition with the
// given type and status "True", or nil if not found.
func conditionTime(conditions []interface{}, condType string) *time.Time {
	for _, c := range conditions {
		cm, _ := c.(map[string]interface{})
		if cm["type"] == condType && cm["status"] == "True" {
			if ts, ok := cm["lastTransitionTime"].(string); ok {
				if t, err := time.Parse(time.RFC3339, ts); err == nil {
					return &t
				}
			}
		}
	}
	return nil
}

func (m model) pollClusterCmd(idx int, def clusterDef) tea.Cmd {
	client := m.client
	prev := m.states[idx]
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		defer cancel()

		next := clusterState{
			startTime:     prev.startTime,
			cpReady:       milestone{done: prev.cpReady.done, at: prev.cpReady.at},
			workersReady:  milestone{done: prev.workersReady.done, at: prev.workersReady.at},
			rancherActive: milestone{done: prev.rancherActive.done, at: prev.rancherActive.at},
		}

		// CAPI cluster phase
		obj, err := client.Resource(clusterGVR).Namespace(def.capiNS).Get(ctx, def.capiName, metav1.GetOptions{})
		if err != nil {
			return pollResultMsg{idx: idx, state: clusterState{}}
		}
		status, _ := obj.Object["status"].(map[string]interface{})
		next.phase, _ = status["phase"].(string)
		if next.phase == "" {
			next.phase = "Pending"
		}

		// On first discovery of an existing cluster, seed startTime from creationTimestamp.
		discovered := prev.startTime.IsZero()
		if discovered {
			meta, _ := obj.Object["metadata"].(map[string]interface{})
			if ts, ok := meta["creationTimestamp"].(string); ok {
				if t, err := time.Parse(time.RFC3339, ts); err == nil {
					next.startTime = t
				}
			}
		}

		// CP ready — check status.ready or status.initialized on the CP object.
		// On discovery, reconstruct the milestone timestamp from the condition.
		cpObj, err := client.Resource(def.cpGVR).Namespace(def.capiNS).Get(ctx, def.cpName, metav1.GetOptions{})
		if err == nil {
			cpStatus, _ := cpObj.Object["status"].(map[string]interface{})
			ready, _ := cpStatus["ready"].(bool)
			initialized, _ := cpStatus["initialized"].(bool)
			next.cpReady.done = ready || initialized
			if next.cpReady.done && next.cpReady.at == nil {
				conds, _ := cpStatus["conditions"].([]interface{})
				next.cpReady.at = conditionTime(conds, "Ready")
				// k3k nests conditions under clusterStatus
				if next.cpReady.at == nil {
					if cs, ok := cpStatus["clusterStatus"].(map[string]interface{}); ok {
						conds2, _ := cs["conditions"].([]interface{})
						next.cpReady.at = conditionTime(conds2, "Ready")
					}
				}
			}
		}
		// Discard stale timestamps from previous cluster runs.
		if next.cpReady.at != nil && next.cpReady.at.Before(next.startTime) {
			next.cpReady.at = nil
		}
		next.cpReady.reach(prev.cpReady)

		// Workers ready — MachineDeployment readyReplicas >= 1.
		if def.mdName != "" {
			mdObj, err := client.Resource(mdGVR).Namespace(def.capiNS).Get(ctx, def.mdName, metav1.GetOptions{})
			if err == nil {
				mdStatus, _ := mdObj.Object["status"].(map[string]interface{})
				ready := toFloat64(mdStatus["readyReplicas"])
				if ready == 0 {
					ready = toFloat64(mdStatus["availableReplicas"])
				}
				next.workersReady.done = ready >= 1
				if next.workersReady.done && next.workersReady.at == nil {
					conds, _ := mdStatus["conditions"].([]interface{})
					next.workersReady.at = conditionTime(conds, "Available")
				}
			}
		}
		if next.workersReady.at != nil && next.workersReady.at.Before(next.startTime) {
			next.workersReady.at = nil
		}
		next.workersReady.reach(prev.workersReady)

		// Rancher Active — match provisioning cluster by display-name annotation.
		pList, _ := client.Resource(provisioningGVR).Namespace("fleet-default").List(ctx, metav1.ListOptions{})
		if pList != nil {
			for _, item := range pList.Items {
				meta, _ := item.Object["metadata"].(map[string]interface{})
				ann, _ := meta["annotations"].(map[string]interface{})
				if ann["provisioning.cattle.io/management-cluster-display-name"] == def.capiName {
					st, _ := item.Object["status"].(map[string]interface{})
					next.rancherActive.done, _ = st["ready"].(bool)
					if next.rancherActive.done && next.rancherActive.at == nil {
						conds, _ := st["conditions"].([]interface{})
						next.rancherActive.at = conditionTime(conds, "Ready")
					}
					break
				}
			}
		}
		if next.rancherActive.at != nil && next.rancherActive.at.Before(next.startTime) {
			next.rancherActive.at = nil
		}
		// Rancher provisioning clusters often have no lastTransitionTime on their conditions.
		// Fall back to the CAPI cluster's own Ready condition, which is stable across restarts.
		if next.rancherActive.done && next.rancherActive.at == nil {
			conds, _ := status["conditions"].([]interface{})
			next.rancherActive.at = conditionTime(conds, "Ready")
			if next.rancherActive.at != nil && next.rancherActive.at.Before(next.startTime) {
				next.rancherActive.at = nil
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

		return pollResultMsg{idx: idx, state: next, discovered: discovered}
	}
}

func applySelectedCmd(selected [3]bool) tea.Cmd {
	return func() tea.Msg {
		for i, def := range defs {
			if !selected[i] {
				continue
			}
			for _, f := range def.applyFiles {
				exec.Command("kubectl", "--context", "ranchero-k3s", "apply", "-f", f).Run()
			}
		}
		return applyDoneMsg{}
	}
}

func deleteClusterCmd(idx int, client dynamic.Interface, def clusterDef) tea.Cmd {
	return func() tea.Msg {
		deleteOneCluster(context.Background(), client, def)
		return clusterDeletedMsg{idx: idx}
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
		case "1", "2", "3":
			if m.phase == phaseIdle || m.phase == phaseRunning {
				i := int(msg.String()[0] - '1')
				if !m.deleting[i] {
					m.selected[i] = !m.selected[i]
				}
			}
		case "s":
			switch m.phase {
			case phaseIdle:
				if m.selected[0] || m.selected[1] || m.selected[2] {
					m.phase = phaseConfirm
				}
			case phaseRunning:
				if m.hasPendingAction() {
					m.phase = phaseConfirm
				}
			case phaseConfirm:
				m.phase = phaseRunning
				now := time.Now()
				var toApply [3]bool
				var cmds []tea.Cmd
				for i := range m.states {
					if m.selected[i] && m.states[i].startTime.IsZero() {
						m.states[i].startTime = now
						m.states[i].phase = "Pending"
						toApply[i] = true
					}
					if !m.selected[i] && !m.states[i].startTime.IsZero() && !m.deleting[i] {
						m.deleting[i] = true
						cmds = append(cmds, deleteClusterCmd(i, m.client, defs[i]))
					}
				}
				if toApply[0] || toApply[1] || toApply[2] {
					cmds = append(cmds, applySelectedCmd(toApply))
				}
				return m, tea.Batch(cmds...)
			}
		case "esc":
			if m.phase == phaseConfirm {
				anyRunning := false
				for i := range m.states {
					if !m.states[i].startTime.IsZero() || m.deleting[i] {
						anyRunning = true
					}
				}
				if anyRunning {
					m.phase = phaseRunning
				} else {
					m.phase = phaseIdle
				}
			}
		}

	case tickMsg:
		return m, tea.Batch(tickCmd(), m.pollAllCmd())

	case applyDoneMsg:
		// nothing; poll loop picks up state changes

	case pollResultMsg:
		m.states[msg.idx] = msg.state
		if msg.discovered && m.phase == phaseIdle {
			m.phase = phaseRunning
		}

	case clusterDeletedMsg:
		m.deleting[msg.idx] = false
		m.states[msg.idx] = clusterState{}
		m.selected[msg.idx] = true
		anyActive := false
		for i := range m.states {
			if !m.states[i].startTime.IsZero() || m.deleting[i] {
				anyActive = true
			}
		}
		if !anyActive {
			m.phase = phaseIdle
		}
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

		numKey := dimSt.Render(fmt.Sprintf("[%d]", i+1))
		content := titleSt.Render(def.label) + " " + numKey + "\n" +
			subtitleSt.Render(def.subtitle) + "\n\n" +
			statusLabel(s, noWorkers) + "  " + timerStr + "\n\n" +
			ms1 + "\n" +
			ms2 + "\n" +
			ms3 + "\n\n" +
			cpuLine + "\n" +
			memLine + "\n"

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

	return "\n" + row + "\n\n" + help + "\n"
}

// ─── delete mode ─────────────────────────────────────────────────────────────

var jsonRemoveFinalizers = []byte(`[{"op":"remove","path":"/metadata/finalizers"}]`)

func resourceIface(client dynamic.Interface, gvr schema.GroupVersionResource, ns string) dynamic.ResourceInterface {
	if ns == "" {
		return client.Resource(gvr)
	}
	return client.Resource(gvr).Namespace(ns)
}

func forceClearFinalizers(ctx context.Context, client dynamic.Interface, gvr schema.GroupVersionResource, ns, name string) {
	iface := resourceIface(client, gvr, ns)
	obj, err := iface.Get(ctx, name, metav1.GetOptions{})
	if err != nil || len(obj.GetFinalizers()) == 0 {
		return
	}
	iface.Patch(ctx, name, types.JSONPatchType, jsonRemoveFinalizers, metav1.PatchOptions{}) //nolint:errcheck
}

func findRancherCluster(ctx context.Context, client dynamic.Interface, capiName string) (provName, mgmtName string) {
	pList, err := client.Resource(provisioningGVR).Namespace("fleet-default").List(ctx, metav1.ListOptions{})
	if err != nil || pList == nil {
		return
	}
	for _, item := range pList.Items {
		meta, _ := item.Object["metadata"].(map[string]interface{})
		ann, _ := meta["annotations"].(map[string]interface{})
		if ann["provisioning.cattle.io/management-cluster-display-name"] != capiName {
			continue
		}
		provName, _ = meta["name"].(string)
		st, _ := item.Object["status"].(map[string]interface{})
		mgmtName, _ = st["clusterName"].(string)
		return
	}
	return
}

func deleteOneCluster(ctx context.Context, client dynamic.Interface, def clusterDef) {
	nsGVR := schema.GroupVersionResource{Version: "v1", Resource: "namespaces"}

	// Delete the Rancher management cluster first — if GC cascades to the provisioning
	// cluster we get clean teardown for free; CAPI namespace follows after.
	provName, mgmtName := findRancherCluster(ctx, client, def.capiName)
	deleteRancherCluster(ctx, client, def.capiName, provName, mgmtName)

	forceClearFinalizers(ctx, client, clusterGVR, def.capiNS, def.capiName)
	forceClearFinalizers(ctx, client, def.cpGVR, def.capiNS, def.cpName)

	client.Resource(nsGVR).Delete(ctx, def.capiNS, metav1.DeleteOptions{}) //nolint:errcheck
	fmt.Printf("deleting %s\n", def.capiNS)

	for {
		_, err := client.Resource(nsGVR).Get(ctx, def.capiNS, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			fmt.Printf("done:     %s\n", def.capiNS)
			break
		}
		// Turtles and other controllers re-add finalizers during teardown; keep clearing.
		forceClearFinalizers(ctx, client, clusterGVR, def.capiNS, def.capiName)
		forceClearFinalizers(ctx, client, def.cpGVR, def.capiNS, def.cpName)
		time.Sleep(500 * time.Millisecond)
	}

	if def.extraNamespace != "" {
		client.Resource(nsGVR).Delete(ctx, def.extraNamespace, metav1.DeleteOptions{}) //nolint:errcheck
		fmt.Printf("deleted  %s\n", def.extraNamespace)
	}
}

func deleteRancherCluster(ctx context.Context, client dynamic.Interface, capiName, provName, mgmtName string) {
	if mgmtName == "" || mgmtName == "local" {
		return
	}

	// Strip finalizers and delete the management cluster. If owner references are in
	// place, GC will cascade to the provisioning cluster automatically.
	forceClearFinalizers(ctx, client, managementClusterGVR, "", mgmtName)
	client.Resource(managementClusterGVR).Delete(ctx, mgmtName, metav1.DeleteOptions{}) //nolint:errcheck
	fmt.Printf("deleting management/%s (%s)\n", mgmtName, capiName)

	for {
		_, err := client.Resource(managementClusterGVR).Get(ctx, mgmtName, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			fmt.Printf("done:     management/%s\n", mgmtName)
			break
		}
		forceClearFinalizers(ctx, client, managementClusterGVR, "", mgmtName)
		time.Sleep(500 * time.Millisecond)
	}

	// Check whether GC cascaded to the provisioning cluster; if not, clean it up manually.
	if provName == "" {
		return
	}
	_, err := client.Resource(provisioningGVR).Namespace("fleet-default").Get(ctx, provName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		fmt.Printf("done:     rancher/%s (cascaded)\n", provName)
		return
	}
	forceClearFinalizers(ctx, client, provisioningGVR, "fleet-default", provName)
	client.Resource(provisioningGVR).Namespace("fleet-default").Delete(ctx, provName, metav1.DeleteOptions{}) //nolint:errcheck
	fmt.Printf("deleting rancher/%s (%s)\n", provName, capiName)
	for {
		_, err := client.Resource(provisioningGVR).Namespace("fleet-default").Get(ctx, provName, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			fmt.Printf("done:     rancher/%s\n", provName)
			return
		}
		forceClearFinalizers(ctx, client, provisioningGVR, "fleet-default", provName)
		time.Sleep(500 * time.Millisecond)
	}
}

func runDelete(client dynamic.Interface) {
	ctx := context.Background()
	var wg sync.WaitGroup
	for _, def := range defs {
		wg.Add(1)
		go func(def clusterDef) {
			defer wg.Done()
			deleteOneCluster(ctx, client, def)
		}(def)
	}
	wg.Wait()
	fmt.Println("all clusters deleted")
}

// ─── main ─────────────────────────────────────────────────────────────────────

func main() {
	klog.SetOutput(io.Discard)

	if len(os.Args) > 1 && os.Args[1] == "delete" {
		runDelete(newModel().client)
		return
	}

	p := tea.NewProgram(newModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
