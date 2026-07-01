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
      <img :src="$base + 'Manuel_Solingen_g.png'" class="about-photo" />
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
<img :src="$base + 'diagrams/shared-cluster.svg'" class="w-full h-full object-contain" />
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
## Isolation
- Cluster-scoped CRDs & RBAC
- One team's mistake = everyone's blast radius

::box2::
## Operations
- Upgrade cycles are coupled
- Backup restores affect all tenants

::box3::
## Infrastructure
- Node resources compete directly
- Network & storage boundaries are weak

---
layout: centered
---

## We Solved Isolation…
# …by Creating More Clusters

*Strong isolation comes at the cost of operational complexity.*

---
layout: default
---

# Notice Something?

<div class="notice-grid">
  <div class="notice-col">
    <div class="notice-year">2018</div>
    <div class="notice-item app">Application</div>
    <div class="notice-arrow">↓</div>
    <div class="notice-item cluster">Cluster</div>
  </div>
  <div class="notice-col">
    <div class="notice-year">2026</div>
    <div class="notice-item app">Application</div>
    <div class="notice-arrow">↓</div>
    <div class="notice-item cluster">Cluster</div>
    <div class="notice-arrow">↓</div>
    <div class="notice-item mgmt">Management Cluster</div>
    <div class="notice-arrow">↓</div>
    <div class="notice-item hcp">Hosted Control Plane</div>
    <div class="notice-arrow">↓</div>
    <div class="notice-item virt">Virtualization <span class="notice-note">(if using VMs)</span></div>
    <div class="notice-arrow">↓</div>
    <div class="notice-item metal">Bare Metal</div>
  </div>
</div>

<p class="notice-punchline">We're now building Kubernetes platforms that deploy Kubernetes platforms.</p>

<style>
.notice-grid {
  display: flex;
  gap: 6rem;
  justify-content: center;
  align-items: flex-start;
  margin-top: 1rem;
}
.notice-col {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.2rem;
}
.notice-year {
  font-size: 1.3rem;
  font-weight: 900;
  margin-bottom: 0.5rem;
  color: var(--slidev-theme-primary);
}
.notice-item {
  padding: 0.35rem 1.2rem;
  border-radius: 4px;
  font-size: 0.85rem;
  font-weight: 600;
  text-align: center;
  min-width: 180px;
}
.notice-arrow { font-size: 1rem; color: #999; }
.app    { background: #f0f0f0; }
.cluster { background: #d0eaf4; }
.mgmt   { background: #a8d5e8; }
.hcp    { background: var(--slidev-theme-primary); color: #fff; }
.virt   { background: #e8e8f4; }
.metal  { background: #ddd; }
.notice-note { font-size: 0.7rem; font-weight: 400; opacity: 0.8; }
.notice-punchline {
  text-align: center;
  margin-top: 1.2rem;
  font-style: italic;
  font-size: 0.95rem;
  color: #444;
}
</style>

---
layout: section-simple
---

# The Evolution of the Control Plane

> We no longer deploy applications. We deploy Kubernetes.

---
layout: default
---

# The Evolution of Isolation

<div class="spectrum-row">
  <div class="spec-item">
    <div class="spec-box spec-ns">Namespaces</div>
    <div class="spec-sub">shared cluster</div>
  </div>
  <div class="spec-item" v-click>
    <div class="spec-connector">→</div>
    <div class="spec-box spec-vc">Virtual Clusters</div>
    <div class="spec-sub">per-tenant API</div>
  </div>
  <div class="spec-item" v-click>
    <div class="spec-connector">→</div>
    <div class="spec-box spec-hcp">Hosted Control Planes</div>
    <div class="spec-sub">CP as Pods</div>
  </div>
  <div class="spec-item" v-click>
    <div class="spec-connector">→</div>
    <div class="spec-box spec-ded">Dedicated Clusters</div>
    <div class="spec-sub">full isolation</div>
  </div>
</div>
<div class="spec-legend">
  <span>← lower isolation · lower cost · higher density</span>
  <span>higher isolation · higher cost · lower density →</span>
</div>

<style>
.spectrum-row {
  display: flex;
  align-items: flex-start;
  justify-content: center;
  gap: 0;
  margin-top: 2rem;
}
.spec-item {
  display: flex;
  align-items: center;
  gap: 0;
}
.spec-box {
  padding: 1rem 1.4rem;
  border-radius: 6px;
  font-weight: 700;
  font-size: 0.95rem;
  text-align: center;
  min-width: 150px;
}
.spec-ns  { background: #e8f4f8; }
.spec-vc  { background: #d0eaf4; }
.spec-hcp { background: #a8d5e8; }
.spec-ded { background: #7bbfd8; }
.spec-sub {
  text-align: center;
  font-size: 0.72rem;
  color: #777;
  margin-top: 0.4rem;
}
.spec-connector {
  font-size: 1.5rem;
  color: #bbb;
  padding: 0 0.6rem;
  padding-bottom: 1.2rem;
}
.spec-item { flex-direction: column; align-items: center; }
.spec-item:not(:first-child) { flex-direction: row; align-items: flex-start; }
.spec-legend {
  display: flex;
  justify-content: space-between;
  font-size: 0.75rem;
  color: #aaa;
  border-top: 1px solid #e0e0e0;
  margin-top: 2rem;
  padding-top: 0.5rem;
}
</style>

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
✅ Worker pools can be shared or dedicated per tenant

❌ More complexity than virtual clusters  
❌ Operational overhead of managing many control planes

> *Control planes become cattle, not pets.*

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
❌ Every cluster becomes a product to operate

---
layout: section-simple
---

# Choosing the Right Model

---
layout: default
---

# There Is No "Best"

<table class="tradeoff-table">
  <thead><tr><th>Model</th><th>Isolation</th><th>Cost</th><th>Density</th><th>Best For</th></tr></thead>
  <tbody>
    <tr class="row-ns"><td>Namespaces</td><td>★☆☆☆☆</td><td>$</td><td>★★★★★</td><td>Internal teams</td></tr>
    <tr class="row-vc"><td>Virtual Clusters</td><td>★★★☆☆</td><td>$$</td><td>★★★★☆</td><td>Dev environments</td></tr>
    <tr class="row-hcp"><td>Hosted Control Planes</td><td>★★★★☆</td><td>$$$</td><td>★★★☆☆</td><td>SaaS, edge</td></tr>
    <tr class="row-ded"><td>Dedicated Clusters</td><td>★★★★★</td><td>$$$$</td><td>★★☆☆☆</td><td>Regulated, GPU</td></tr>
  </tbody>
</table>

*Every model optimizes for a different trade-off.*

<style>
.tradeoff-table { width: 100%; border-collapse: collapse; font-size: 0.9rem; margin-bottom: 0.75rem; }
.tradeoff-table th { padding: 0.5rem 0.75rem; text-align: left; border-bottom: 2px solid #ccc; font-weight: 700; }
.tradeoff-table td { padding: 0.5rem 0.75rem; }
.row-ns  { background: #e8f4f8; }
.row-vc  { background: #d0eaf4; }
.row-hcp { background: #a8d5e8; }
.row-ded { background: #7bbfd8; }
</style>

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

<img :src="$base + 'diagrams/hcp-models.svg'" class="h-4/5 mx-auto" />

<!--
In production this would typically be a Harvester HCI cluster instead of bare k3s — same KubeVirt foundation, but with proper storage, networking, and a management UI built in.
-->

---
layout: boxes-green-3
---

# One Management Cluster, Three Models

A single bare-metal k3s cluster running everything.

::box1::
## KubeVirt
Control plane VM + Worker VMs
*Dedicated CP + Dedicated Workers*

::box2::
## Kamaji + KubeVirt
Control plane Pods + Worker VMs
*Hosted CP + Dedicated Workers*

::box3::
## k3k
k3s Pods + Shared workers
*Hosted CP + Shared Workers*

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
layout: centered
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
- The future isn't managing clusters — it's managing **fleets of control planes**

---
layout: default
class: contact-slide
---

<div class="contact-layout">
  <div class="contact-left">
    <h1>Thank You</h1>
    <p class="contact-subtitle">Let's keep the conversation going.</p>
    <div class="contact-links">
      <div class="contact-row">
        <span class="contact-icon">𝕏</span>
        <span>x.com/manuel_zapf</span>
      </div>
      <div class="contact-row">
        <span class="contact-icon">in</span>
        <span>linkedin.com/in/manuel-zapf-374a4869</span>
      </div>
      <div class="contact-row">
        <span class="contact-icon">gh</span>
        <span>github.com/SantoDE</span>
      </div>
      <div class="contact-row">
        <span class="contact-icon">✉</span>
        <span>manuel.zapf@codecentric.de</span>
      </div>
    </div>
  </div>
  <div class="contact-right">
    <img :src="$base + 'qr-slides.svg'" class="contact-qr" />
    <p class="contact-qr-label">slides.manuelzapf.io/from-clusters-to-controlplanes</p>
  </div>
</div>
