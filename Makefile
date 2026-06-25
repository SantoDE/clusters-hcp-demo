BINARY        := /home/manuelzapf/Projects/ranchero/bin/ranchero-linux-amd64
DEPLOY_IP     ?= 51.195.5.176
DEPLOY_HOST   ?= ubuntu@$(DEPLOY_IP)
DEPLOY_PATH   ?= /usr/local/bin/ranchero
REMOTE_CONFIG ?= config.toml
KUBECONFIG_OUT ?= kubeconfig.yaml
FLEET_REPO    ?= https://github.com/SantoDE/clusters-hcp-demo

.PHONY: deploy run kubeconfig fleet-apply

deploy:
	scp $(BINARY) $(DEPLOY_HOST):/tmp/ranchero
	ssh $(DEPLOY_HOST) "sudo mv /tmp/ranchero $(DEPLOY_PATH) && sudo chmod +x $(DEPLOY_PATH)"

run:
	scp rancher-config.yaml $(DEPLOY_HOST):~/rancher-config.yaml
	ssh $(DEPLOY_HOST) "sudo ranchero --config $(REMOTE_CONFIG)"

kubeconfig:
	ssh $(DEPLOY_HOST) "sudo cat /etc/rancher/k3s/k3s.yaml" | sed 's|https://127.0.0.1|https://$(DEPLOY_IP)|g' > $(KUBECONFIG_OUT)
	kubectl --kubeconfig=$(KUBECONFIG_OUT) config rename-context default ranchero-k3s
	KUBECONFIG=$(KUBECONFIG_OUT):$$HOME/.kube/config kubectl config view --flatten > /tmp/merged-kubeconfig.yaml
	mv /tmp/merged-kubeconfig.yaml $$HOME/.kube/config
	@echo "merged into ~/.kube/config"

fleet-apply:
	sed 's|FLEET_REPO_PLACEHOLDER|$(FLEET_REPO)|' gitrepos/local.yaml | kubectl --context ranchero-k3s apply -f -
