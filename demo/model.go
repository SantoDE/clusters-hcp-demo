package main

import (
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
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
	phase     string
	pods      []podInfo
	totalCPU  int64
	totalMem  int64
	startTime time.Time

	cpReady       milestone
	workersReady  milestone
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
