package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
)

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
