package main

import (
	"fmt"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

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
