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

<div class="about-card">
<div class="about-grid">
  <div class="about-left">
    <div class="photo-frame">
      <img :src="'/Manuel_Solingen_g.png'" class="about-photo" />
    </div>
    <div class="about-name">Manuel Zapf</div>
    <div class="about-title">Principal Solution Architect</div>
    <ul class="about-bio">
      <li>Traefik Maintainer</li>
      <li>Dapr Meteor</li>
      <li>Proud dad</li>
      <li>A bit too much into Handball</li>
      <li>Previously: Traefik Labs, Solo.io</li>
    </ul>
  </div>
  <div class="about-right">
    <div class="skills-title">Skills &amp; Tools</div>
    <div class="skills-tags">
      <span class="tag">Kubernetes</span>
      <span class="tag">Cloud Native Development</span>
      <span class="tag">API Gateways</span>
      <span class="tag">Service Meshes</span>
      <span class="tag">Architecture</span>
    </div>
    <div class="social-links">
      <div class="social-link">
        <span class="social-icon social-x">𝕏</span>
        <span>https://x.com/manuel_zapf</span>
      </div>
      <div class="social-link">
        <span class="social-icon social-li">in</span>
        <span>https://www.linkedin.com/in/manuel-zapf-374a4869/</span>
      </div>
      <div class="social-link">
        <span class="social-icon social-gh">GH</span>
        <span>https://github.com/SantoDE</span>
      </div>
    </div>
  </div>
</div>
</div>

<style>
.about-card {
  background: #f0f0f0;
  border-radius: 8px;
  padding: 1.5rem 2rem;
  height: 88%;
  display: flex;
  align-items: stretch;
}
.about-grid {
  display: grid;
  grid-template-columns: 40% 56%;
  gap: 4%;
  width: 100%;
  align-items: start;
}
.about-left {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
}
.photo-frame {
  width: 200px;
  height: 200px;
  margin-bottom: 0.6rem;
}
.about-photo {
  width: 100%;
  height: 100%;
  object-fit: cover;
  border-radius: 4px;
}
.about-name {
  font-size: 1.4rem;
  font-weight: 900;
  font-style: italic;
  line-height: 1.2;
}
.about-title {
  font-size: 1rem;
  font-style: italic;
  font-weight: 600;
  margin-bottom: 0.4rem;
  color: #444;
}
.about-bio {
  list-style: none;
  padding: 0;
  margin: 0;
  font-size: 0.85rem;
  line-height: 1.55;
}
.about-bio li::before {
  content: '●';
  color: var(--slidev-theme-primary);
  margin-right: 0.5rem;
}
.skills-title {
  font-size: 1.3rem;
  font-weight: 800;
  margin-bottom: 0.75rem;
}
.skills-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin-bottom: 1.5rem;
}
.tag {
  border: 2px solid var(--slidev-theme-primary);
  border-radius: 4px;
  padding: 0.25rem 0.75rem;
  font-size: 0.85rem;
  font-weight: 600;
}
.social-links {
  display: flex;
  flex-direction: column;
  gap: 0.6rem;
}
.social-link {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  font-size: 0.8rem;
}
.social-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 2rem;
  height: 2rem;
  border-radius: 6px;
  font-weight: 900;
  font-size: 1rem;
  flex-shrink: 0;
}
.social-x   { background: #000; color: #fff; }
.social-li  { background: #0a66c2; color: #fff; font-size: 0.85rem; }
.social-gh  { background: #24292f; color: #fff; font-size: 0.65rem; font-weight: 900; letter-spacing: 0; }
</style>

---
layout: longtext-left
image: /Falcon_Heavy_Demo_Mission_(39337245145).jpg
class: agenda-slide
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

---
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
class: diagram-slide
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
class: diagram-slide
---

# Virtual Clusters

**e.g. k3k · vCluster**

✅ Separate Kubernetes API per tenant  
✅ Fast provisioning  
✅ Great for development and SaaS

❌ Workloads still share worker nodes  
❌ Weaker isolation than a dedicated control plane

---
layout: longtext-left
image: /diagrams/hosted-cp.svg
class: diagram-slide
---

# Hosted Control Planes

**e.g. Kamaji · HyperShift**

✅ Dedicated control plane per tenant (runs as Pods)  
✅ Independent lifecycle — upgrade without coordination  
✅ Shared worker infrastructure keeps cost down

❌ More complexity than virtual clusters  
❌ Workers still shared — blast radius remains

---
layout: longtext-left
image: /diagrams/dedicated.svg
class: diagram-slide
---

# Dedicated Clusters

**e.g. Cloud Managed · Bare Metal**

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
| Hosted Control Planes | ★★★★☆ | $$$ | ★★★☆☆ | SaaS, edge, Scall |
| Dedicated Clusters | ★★★★★ | $$$$ | ★★☆☆☆ | Regulated, GPU |

*Every model optimizes for a different trade-off.*

---
layout: default
---

# Choosing Your Control Plane Model

| Requirement | Model | Examples |
|---|---|---|
| Internal teams, cost-sensitive | Namespaces | Native Kubernetes |
| Developer & CI environments | Virtual Clusters | vCluster, k3k |
| SaaS, multi-tenant platforms & Scale| Hosted Control Planes | Kamaji, k3k, HyperShift |
| Regulated workloads, GPU, compliance | Dedicated Clusters | Any infra provider |

---
layout: cover
image: /examples/write_sticker_1.jpg
---

# Live Demo

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
