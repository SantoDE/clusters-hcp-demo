BINARY        := /home/manuelzapf/Projects/ranchero/bin/ranchero-linux-amd64
DEPLOY_IP     ?= 51.195.5.176
DEPLOY_HOST   ?= ubuntu@$(DEPLOY_IP)
DEPLOY_PATH   ?= /usr/local/bin/ranchero
REMOTE_CONFIG ?= config.toml
KUBECONFIG_OUT ?= kubeconfig.yaml
FLEET_REPO    ?= https://github.com/SantoDE/clusters-hcp-demo
KUBECTL       ?= kubectl --context ranchero-k3s

.PHONY: deploy run run-force kubeconfig fleet-apply \
        clusters-apply clusters-delete clusters-reset demo

deploy:
	scp $(BINARY) $(DEPLOY_HOST):/tmp/ranchero
	ssh $(DEPLOY_HOST) "sudo mv /tmp/ranchero $(DEPLOY_PATH) && sudo chmod +x $(DEPLOY_PATH)"

run:
	scp rancher-config.yaml $(DEPLOY_HOST):~/$(REMOTE_CONFIG)
	ssh $(DEPLOY_HOST) "sudo ranchero --config $(REMOTE_CONFIG)"
	$(MAKE) kubeconfig
	$(MAKE) fleet-apply

run-force:
	scp rancher-config.yaml $(DEPLOY_HOST):~/$(REMOTE_CONFIG)
	ssh $(DEPLOY_HOST) "sudo ranchero --config $(REMOTE_CONFIG) --force"
	$(MAKE) kubeconfig
	$(MAKE) fleet-apply

kubeconfig:
	ssh $(DEPLOY_HOST) "sudo cat /etc/rancher/k3s/k3s.yaml" | sed 's|https://127.0.0.1|https://$(DEPLOY_IP)|g' > $(KUBECONFIG_OUT)
	kubectl --kubeconfig=$(KUBECONFIG_OUT) config rename-context default ranchero-k3s
	KUBECONFIG=$(KUBECONFIG_OUT):$$HOME/.kube/config kubectl config view --flatten > /tmp/merged-kubeconfig.yaml
	mv /tmp/merged-kubeconfig.yaml $$HOME/.kube/config
	@echo "merged into ~/.kube/config"

fleet-apply:
	sed 's|FLEET_REPO_PLACEHOLDER|$(FLEET_REPO)|' gitrepos/local.yaml | kubectl --context ranchero-k3s apply -f -

clusters-apply:
	$(KUBECTL) apply -f clusters/k3s-kubevirt/cluster.yaml
	$(KUBECTL) apply -f clusters/kamaji-kubevirt/cluster.yaml
	$(KUBECTL) apply -f clusters/kamaji-kubevirt/cni-configmap.yaml
	$(KUBECTL) apply -f clusters/kamaji-kubevirt/cni.yaml
	$(KUBECTL) apply -f clusters/k3k/provider.yaml
	$(KUBECTL) apply -f clusters/k3k/cluster.yaml

clusters-delete:
	-$(KUBECTL) patch cluster k3s-kubevirt -n capi-k3s-kubevirt --type=json \
		-p='[{"op":"remove","path":"/metadata/finalizers"}]' 2>/dev/null
	-$(KUBECTL) patch cluster kamaji-kubevirt -n capi-kamaji-kubevirt --type=json \
		-p='[{"op":"remove","path":"/metadata/finalizers"}]' 2>/dev/null
	-$(KUBECTL) patch cluster k3k-simple -n capi-k3k --type=json \
		-p='[{"op":"remove","path":"/metadata/finalizers"}]' 2>/dev/null
	$(KUBECTL) delete -f clusters/k3s-kubevirt/cluster.yaml --ignore-not-found
	$(KUBECTL) delete -f clusters/kamaji-kubevirt/cni.yaml --ignore-not-found
	$(KUBECTL) delete -f clusters/kamaji-kubevirt/cni-configmap.yaml --ignore-not-found
	$(KUBECTL) delete -f clusters/kamaji-kubevirt/cluster.yaml --ignore-not-found
	$(KUBECTL) delete -f clusters/k3k/cluster.yaml --ignore-not-found

clusters-reset: clusters-delete clusters-apply

demo:
	nix run .#demo
