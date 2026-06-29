package main

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

// ─── cluster definitions ──────────────────────────────────────────────────────

type clusterDef struct {
	label     string
	subtitle  string
	namespace string
	name      string
}

var clusters = []clusterDef{
	{
		label:     "KubeVirt k3s",
		subtitle:  "VMs all the way down",
		namespace: "capi-k3s-kubevirt",
		name:      "k3s-kubevirt",
	},
	{
		label:     "Kamaji + KubeVirt",
		subtitle:  "Hosted CP, VM workers",
		namespace: "capi-kamaji-kubevirt",
		name:      "kamaji-kubevirt",
	},
	{
		label:     "k3k",
		subtitle:  "Pure pods, no VMs",
		namespace: "capi-k3k",
		name:      "k3k-simple",
	},
}

// ─── state ────────────────────────────────────────────────────────────────────

type clusterState struct {
	phase     string
	available bool
	ready     bool
	events    []string
	startTime time.Time
	doneTime  *time.Time
}

type model struct {
	client   dynamic.Interface
	states   [3]clusterState
	tick     int
	started  bool
	width    int
	height   int
}

// ─── messages ─────────────────────────────────────────────────────────────────

type tickMsg time.Time
type stateMsg struct {
	idx   int
	state clusterState
}

// ─── k8s GVRs ─────────────────────────────────────────────────────────────────

var clusterGVR = schema.GroupVersionResource{
	Group:    "cluster.x-k8s.io",
	Version:  "v1beta1",
	Resource: "clusters",
}

var provisioningGVR = schema.GroupVersionResource{
	Group:    "provisioning.cattle.io",
	Version:  "v1",
	Resource: "clusters",
}

// ─── init ─────────────────────────────────────────────────────────────────────

func newModel() model {
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		home, _ := os.UserHomeDir()
		kubeconfigPath = home + "/.kube/config"
	}
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "kubeconfig error: %v\n", err)
		os.Exit(1)
	}
	client, err := dynamic.NewForConfig(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "client error: %v\n", err)
		os.Exit(1)
	}

	now := time.Now()
	var states [3]clusterState
	for i := range states {
		states[i] = clusterState{
			phase:     "Pending",
			startTime: now,
			events:    []string{},
		}
	}

	return model{client: client, states: states}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(tick(), m.pollAll())
}

// ─── update ───────────────────────────────────────────────────────────────────

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		m.tick++
		return m, tea.Batch(tick(), m.pollAll())

	case stateMsg:
		prev := m.states[msg.idx]
		next := msg.state

		// capture done time
		if !prev.ready && next.ready && next.doneTime == nil {
			t := time.Now()
			next.doneTime = &t
		} else if prev.doneTime != nil {
			next.doneTime = prev.doneTime
		}

		// append event log on phase change
		next.events = prev.events
		if prev.phase != next.phase && next.phase != "" {
			next.events = append(next.events, fmt.Sprintf("%s → %s", prev.phase, next.phase))
			if len(next.events) > 6 {
				next.events = next.events[len(next.events)-6:]
			}
		}
		next.startTime = prev.startTime
		m.states[msg.idx] = next
	}
	return m, nil
}

// ─── poll ─────────────────────────────────────────────────────────────────────

func (m model) pollAll() tea.Cmd {
	return func() tea.Msg {
		return nil
	}
}

func pollCluster(client dynamic.Interface, idx int, def clusterDef) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		obj, err := client.Resource(clusterGVR).Namespace(def.namespace).Get(ctx, def.name, metav1.GetOptions{})
		if err != nil {
			return stateMsg{idx: idx, state: clusterState{phase: "Not found", events: []string{}}}
		}

		status, _ := obj.Object["status"].(map[string]interface{})
		phase, _ := status["phase"].(string)
		if phase == "" {
			phase = "Pending"
		}

		// check Available condition
		available := false
		if conditions, ok := status["conditions"].([]interface{}); ok {
			for _, c := range conditions {
				cond, _ := c.(map[string]interface{})
				if cond["type"] == "Available" && cond["status"] == "True" {
					available = true
				}
				if cond["type"] == "Ready" && cond["status"] == "True" {
					_ = true // Ready condition tracked via rancher provisioning
				}
			}
		}

		// check rancher provisioning ready
		rancherReady := false
		pList, err2 := client.Resource(provisioningGVR).Namespace("fleet-default").List(ctx, metav1.ListOptions{})
		if err2 == nil {
			for _, item := range pList.Items {
				st, _ := item.Object["status"].(map[string]interface{})
				if r, _ := st["ready"].(bool); r {
					rancherReady = true
					break
				}
			}
		}

		return stateMsg{
			idx: idx,
			state: clusterState{
				phase:     phase,
				available: available,
				ready:     available && rancherReady,
				events:    []string{},
			},
		}
	}
}

func tick() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// ─── styles ───────────────────────────────────────────────────────────────────

var (
	colWidth = 36

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("62")).
			Padding(0, 1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Italic(true)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1, 2).
			Width(colWidth)

	panelActiveStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("82")).
				Padding(1, 2).
				Width(colWidth)

	phaseStyle = lipgloss.NewStyle().Bold(true)

	timerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))

	doneTimerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)

	eventStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))

	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	statusDot = map[bool]string{true: "●", false: "○"}
)

func phaseColor(phase string) lipgloss.Style {
	switch phase {
	case "Provisioned", "Provisioning":
		return phaseStyle.Foreground(lipgloss.Color("214"))
	case "Not found", "Pending":
		return phaseStyle.Foreground(lipgloss.Color("240"))
	default:
		return phaseStyle.Foreground(lipgloss.Color("250"))
	}
}

// ─── view ─────────────────────────────────────────────────────────────────────

func (m model) View() string {
	panels := make([]string, 3)

	for i, def := range clusters {
		s := m.states[i]

		elapsed := time.Since(s.startTime).Round(time.Second)
		timerStr := ""
		if s.ready && s.doneTime != nil {
			finalElapsed := s.doneTime.Sub(s.startTime).Round(time.Second)
			timerStr = doneTimerStyle.Render(fmt.Sprintf("✓ %s", finalElapsed))
		} else {
			timerStr = timerStyle.Render(fmt.Sprintf("⏱ %s", elapsed))
		}

		availDot := eventStyle.Render(statusDot[s.available])
		readyDot := eventStyle.Render(statusDot[s.ready])
		if s.available {
			availDot = lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Render("●")
		}
		if s.ready {
			readyDot = lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Render("●")
		}

		evLog := ""
		for _, e := range s.events {
			evLog += eventStyle.Render("  "+e) + "\n"
		}

		content := fmt.Sprintf("%s\n%s\n\n%s  %s\n\n%s CAPI Available\n%s Rancher Active\n\n%s",
			titleStyle.Render(def.label),
			subtitleStyle.Render(def.subtitle),
			phaseColor(s.phase).Render(s.phase),
			timerStr,
			availDot,
			readyDot,
			evLog,
		)

		style := panelStyle
		if s.ready {
			style = panelActiveStyle
		}
		panels[i] = style.Render(content)
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, panels[0], "  ", panels[1], "  ", panels[2])
	help := helpStyle.Render("  q quit")

	return "\n" + row + "\n\n" + help + "\n"
}

// ─── main ─────────────────────────────────────────────────────────────────────

func main() {
	m := newModel()

	// start polling each cluster independently
	p := tea.NewProgram(m, tea.WithAltScreen())

	// override Init to also kick off per-cluster polling
	go func() {
		for {
			time.Sleep(3 * time.Second)
			for i, def := range clusters {
				cmd := pollCluster(m.client, i, def)
				if msg := cmd(); msg != nil {
					p.Send(msg)
				}
			}
		}
	}()

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
