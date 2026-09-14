package main

import (
	"context"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
)

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
				usage := metricsByPod[name]
				next.pods = append(next.pods, podInfo{name: name, status: phase})
				next.totalCPU += usage[0]
				next.totalMem += usage[1]
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
				exec.Command("kubectl", "--context", "ranchero-k3s", "apply", "-f", f).Run() //nolint:errcheck
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
