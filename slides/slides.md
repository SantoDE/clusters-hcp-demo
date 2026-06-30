---
theme: ../../slidev-theme-codecentric
layout: cover
image: /examples/taskboard_3.jpg
---

# From Clusters to Control Planes
## Rethinking Multi-Tenancy at Scale

---
layout: default
---

# Agenda

1. **The Multi-Tenancy Challenge** — why shared clusters hit limits
2. **When Clusters Stop Scaling** — the isolation patterns that emerged
3. **The Evolution of the Control Plane** — a spectrum of models
4. **Choosing the Right Model** — trade-offs in practice
5. **Live Demo** — three ways to provision Kubernetes
6. **Lessons Learned**

---
layout: section-simple
---

# The Multi-Tenancy Challenge

---
layout: default
---

# One Cluster Was Enough…

<figure>
<div class="h-[380px] overflow-hidden">
<img :src="'/diagrams/shared-cluster.svg'" class="w-full h-full object-contain" />
</div>
<figcaption class="text-center text-sm text-gray-400 border-t border-gray-200 pt-2 mt-2">
Namespaces per team · RBAC · NetworkPolicies · Resource Quotas
</figcaption>
</figure>

---
layout: boxes-green-3
---

# …Until It Wasn't

::box1::
## 🏦 Compliance
"I need dedicated infrastructure."

::box2::
## 🤖 AI & GPUs
"I need exclusive GPU access."

::box3::
## 🏢 Enterprise
"I need my own upgrades. I need more control. I need MOAR"

---on
layout: boxes-green-3
---

# What Actually Breaks?

The challenge isn't Kubernetes — it's *sharing* Kubernetes.

::box1::
**Isolation**
- API access
- Security boundaries
- Compliance scope

::box2::
**Operations**
- Independent upgrades
- Kubernetes versions
- Backup & restore

::box3::
**Infrastructure**
- GPU allocation
- Networking
- Storage

---
layout: centered
---

## We Solved Isolation…
# …by Creating More Clusters

*Strong isolation comes at the cost of operational complexity.*

---
layout: section-simple
---

# The Evolution of the Control Plane

> We no longer deploy applications. We deploy Kubernetes.

---
layout: default
---

# The Evolution of Isolation

<figure>
<img :src="'/diagrams/isolation-spectrum.svg'" class="w-full" />
<figcaption class="flex justify-between text-sm text-gray-400 border-t border-gray-200 pt-2 mt-6">
  <span>← lower isolation · lower cost · higher density</span>
  <span>higher isolation · higher cost · lower density →</span>
</figcaption>
</figure>

---
layout: longtext-left
image: /diagrams/namespaces.svg
---

# Namespaces

**The classic shared-cluster model**

✅ Simple, low cost, excellent density  
✅ Native Kubernetes primitives (RBAC, NetworkPolicy, Quotas)

❌ Shared API server, etcd, and CRDs  
❌ Teams upgrade together  
❌ Blast radius spans the entire cluster

---
layout: longtext-left
image: /diagrams/vcluster.svg
---

# Virtual Clusters

**e.g. vCluster**

✅ Separate Kubernetes API per tenant  
✅ Fast provisioning  
✅ Great for development and SaaS

❌ Workloads still share worker nodes  
❌ Weaker isolation than a dedicated control plane

---
layout: longtext-left
image: /diagrams/hosted-cp.svg
---

# Hosted Control Planes

**e.g. Kamaji · k3k · HyperShift**

✅ Dedicated control plane per tenant (runs as Pods)  
✅ Independent lifecycle — upgrade without coordination  
✅ Shared worker infrastructure keeps cost down

❌ More complexity than virtual clusters  
❌ Workers still shared — blast radius remains

---
layout: longtext-left
image: /diagrams/dedicated.svg
---

# Dedicated Clusters

**e.g. Cluster API · KubeVirt · Bare Metal**

✅ Complete isolation — control plane and workers  
✅ Full lifecycle independence  
✅ GPU exclusivity, compliance, custom networking

❌ Highest infrastructure cost  
❌ Highest operational overhead per cluster

---
layout: section-simple
---

# Choosing the Right Model

---
layout: default
---

# There Is No "Best"

| Model | Isolation | Cost | Density | Best For |
|---|---|---|---|---|
| Namespaces | ★☆☆☆☆ | $ | ★★★★★ | Internal teams |
| Virtual Clusters | ★★★☆☆ | $$ | ★★★★☆ | Dev environments |
| Hosted Control Planes | ★★★★☆ | $$$ | ★★★☆☆ | SaaS, edge |
| Dedicated Clusters | ★★★★★ | $$$$ | ★★☆☆☆ | Regulated, GPU |

*Every model optimizes for a different trade-off.*

---
layout: default
---

# Matching Requirements to Models

| Requirement | Model |
|---|---|
| Internal teams | Namespaces |
| Developer environments | vCluster |
| Lightweight edge | k3k |
| SaaS tenants | Kamaji |
| OpenShift fleets | HyperShift |
| Regulated workloads | KubeVirt |
| Performance / GPU | Bare Metal |

---
layout: section-simple
---

# Live Demo

---
layout: cover
image: /examples/write_sticker_1.jpg
---

# Three Ways to Provision Kubernetes
## Same platform. Same API. Three different control plane architectures.

---
layout: default
---

# Demo Architecture

<img :src="'/diagrams/hcp-models.svg'" class="h-4/5 mx-auto" />

---
layout: boxes-green-3
---

# One Management Cluster, Three Models

A single bare-metal k3s cluster running everything.

::box1::
**KubeVirt**

Control plane VM + Worker VMs

*Dedicated Control Plane + Dedicated Workers*

::box2::
**Kamaji + KubeVirt**

Control plane Pods + Worker VMs

*Hosted Control Plane + Dedicated Workers*

::box3::
**k3k**

k3s Pods + Shared workers

*Hosted Control Plane + Shared Workers*

---
layout: default
---

# Comparing the Models

| | KubeVirt | Kamaji + KubeVirt | k3k |
|---|---|---|---|
| Control Plane | VM | Pods | Pods |
| Workers | VM | VM | Shared |
| Isolation | ★★★★★ | ★★★★☆ | ★★☆☆☆ |
| Density | ★★☆☆☆ | ★★★★☆ | ★★★★★ |
| Startup time | Slowest | Fast | Fastest |
| Best fit | Regulated, GPU, Enterprise | SaaS, multi-tenant | Dev, CI, edge |

---
layout: section-simple
---

# It's All About the Control Plane

---
layout: default
---

# Takeaways

- Multi-tenancy is no longer a binary choice between namespaces and clusters
- Modern platforms offer a **spectrum of isolation models**
- The key architectural decision is **where your control plane runs**
- Different workloads require different trade-offs — **there is no universal best**
- Platform engineering is increasingly about managing **fleets of control planes**, not individual workloads

---
layout: cover
image: /examples/taskboard_3.jpg
---

# Thank You
