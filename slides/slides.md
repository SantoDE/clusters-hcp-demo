---
theme: ../../slidev-theme-codecentric
layout: cover
image: /examples/taskboard_3.jpg
---

# Vom Cluster zur Plattform
## Lifecycle, Topology und Multi-Tenancy zusammengedacht

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
    <div class="about-title">Team Lead Private Cloud Solutions @ codecentric cloud</div>
    <ul class="about-bio">
      <li>Traefik Maintainer</li>
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
image: /examples/taskboard_3.jpg
class: agenda-slide
---

# Agenda

1. **The Cluster Proliferation Problem** — why every requirement becomes a new cluster
2. **The Wrong Abstraction Level** — clusters as products vs. resources
3. **HCI as the Foundation** — private cloud built on Hyper-Converged Infrastructure
4. **Clusters as Platform Resources** — one API, three isolation models
5. **GitOps + Self-Service** — how teams actually get clusters
6. **Live Demo**
7. **Lessons Learned**

---
layout: section-simple
---

# The Cluster Proliferation Problem

---
layout: default
---

# Every New Requirement Becomes a New Cluster

<div class="prolif-grid">
  <div class="prolif-item" v-click>
    <div class="prolif-icon">🏦</div>
    <div class="prolif-label">Compliance</div>
    <div class="prolif-sub">needs dedicated infra</div>
  </div>
  <div class="prolif-item" v-click>
    <div class="prolif-icon">🤖</div>
    <div class="prolif-label">AI / GPU</div>
    <div class="prolif-sub">needs exclusive hardware</div>
  </div>
  <div class="prolif-item" v-click>
    <div class="prolif-icon">🏢</div>
    <div class="prolif-label">New Team</div>
    <div class="prolif-sub">wants own upgrade cycle</div>
  </div>
</div>

<div v-click class="prolif-result">
  Platform team: provision, configure, secure, upgrade — repeat.
</div>

<style>
.prolif-grid {
  display: flex;
  justify-content: center;
  gap: 2rem;
  margin: 2rem 0;
}
.prolif-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.4rem;
  background: #f0f0f0;
  border-radius: 8px;
  padding: 1.2rem 1.5rem;
  min-width: 130px;
}
.prolif-icon { font-size: 2rem; }
.prolif-label { font-weight: 700; font-size: 0.95rem; }
.prolif-sub { font-size: 0.75rem; color: #777; text-align: center; }
.prolif-result {
  text-align: center;
  margin-top: 1.5rem;
  font-size: 1.1rem;
  font-style: italic;
  color: #444;
  border-top: 1px solid #ddd;
  padding-top: 1rem;
}
</style>

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
    <div class="notice-item hcp">Platform Operator</div>
    <div class="notice-arrow">↓</div>
    <div class="notice-item virt">Virtualization</div>
    <div class="notice-arrow">↓</div>
    <div class="notice-item metal">Bare Metal</div>
  </div>
</div>

<p class="notice-punchline">We're building Kubernetes platforms that deploy Kubernetes platforms.</p>

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
.notice-punchline {
  text-align: center;
  margin-top: 1.2rem;
  font-style: italic;
  font-size: 0.95rem;
  color: #444;
}
</style>

---
layout: centered
---

## We treat clusters as **products**.

# What if they were **resources**?

*`kubectl apply -f cluster.yaml`*

---
layout: section-simple
---

# HCI as the Foundation

> Hyper-Converged Infrastructure as a standardized private cloud

---
layout: boxes-green-3
---

# What Harvester Gives You

*Harvester — open-source HCI built on KubeVirt, Longhorn, and Kube-OVN*

::box1::
## Compute
KubeVirt VMs with live migration, resource reservations, and GPU passthrough — provisioned via CRs.

::box2::
## Storage
Longhorn distributed block storage. Snapshots, backups, ReadWriteMany — no separate SAN required.

::box3::
## Networking
Kube-OVN overlay networks. Per-cluster subnets, NAT, and LoadBalancer IPs — fully programmable.

---
layout: centered
---

## Same GitOps tools.
## Same API.
# One platform for VMs and Kubernetes.

---
layout: section-simple
---

# Clusters as Platform Resources

---
layout: default
---

# The Isolation Spectrum

<div class="spectrum-row">
  <div class="spec-item">
    <div class="spec-box spec-ns">Virtual Clusters</div>
    <div class="spec-sub">shared workers</div>
  </div>
  <div class="spec-item" v-click>
    <div class="spec-connector">→</div>
    <div class="spec-box spec-vc">Hosted Control Planes</div>
    <div class="spec-sub">dedicated CP + VM workers</div>
  </div>
  <div class="spec-item" v-click>
    <div class="spec-connector">→</div>
    <div class="spec-box spec-ded">Dedicated Clusters</div>
    <div class="spec-sub">full VM isolation</div>
  </div>
</div>
<div class="spec-legend">
  <span>← lower isolation · lower cost · higher density</span>
  <span>higher isolation · higher cost · lower density →</span>
</div>

<div v-click class="spec-punchline">One API shape. The platform decides what runs underneath.</div>

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
  min-width: 170px;
}
.spec-ns  { background: #d0eaf4; }
.spec-vc  { background: #a8d5e8; }
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
.spec-punchline {
  text-align: center;
  margin-top: 1.2rem;
  font-weight: 700;
  font-size: 1rem;
  color: var(--slidev-theme-primary);
}
</style>

---
layout: default
---

# One CRD, Three Isolation Levels

<div class="crd-grid">
<div class="crd-col">

```yaml
apiVersion: platform.codecentric.cloud/v1alpha1
kind: WorkloadCluster
metadata:
  name: team-a
spec:
  type: k3k-shared
  apiHost: team-a.example.com
  network:
    vmCIDR: 10.60.10.0/24
```

<div class="crd-badge badge-shared">virtual cluster · shared workers</div>
</div>
<div class="crd-col">

```yaml
apiVersion: platform.codecentric.cloud/v1alpha1
kind: WorkloadCluster
metadata:
  name: team-b
spec:
  type: k3k-hcp
  apiHost: team-b.example.com
  workers:
    count: 2
  network:
    vmCIDR: 10.60.11.0/24
```

<div class="crd-badge badge-hcp">hosted CP · dedicated VMs</div>
</div>
<div class="crd-col">

```yaml
apiVersion: platform.codecentric.cloud/v1alpha1
kind: WorkloadCluster
metadata:
  name: prod-gpu
spec:
  type: rke2-vm
  apiHost: prod.example.com
  workers:
    count: 3
  network:
    vmCIDR: 10.60.12.0/24
```

<div class="crd-badge badge-rke2">fully dedicated cluster</div>
</div>
</div>

<style>
.crd-grid {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  gap: 0.75rem;
  margin-top: 0.5rem;
}
.crd-col :deep(pre) {
  font-size: 0.58rem !important;
  margin: 0 0 0.4rem;
  line-height: 1.4;
}
.crd-badge {
  text-align: center;
  font-size: 0.7rem;
  font-weight: 600;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
}
.badge-shared { background: #d0eaf4; }
.badge-hcp    { background: #a8d5e8; }
.badge-rke2   { background: #7bbfd8; color: #fff; }
</style>

---
layout: section-simple
---

# GitOps + Self-Service

---
layout: default
---

# From Git Commit to Running Cluster

<div class="gitops-flow">
  <div class="gitops-step" v-click>
    <div class="gitops-icon">📝</div>
    <div class="gitops-label">Developer commits<br/><code>cluster.yaml</code> to Git</div>
  </div>
  <div class="gitops-arrow" v-click>→</div>
  <div class="gitops-step" v-click>
    <div class="gitops-icon">🚢</div>
    <div class="gitops-label">Fleet syncs<br/>WorkloadCluster CR</div>
  </div>
  <div class="gitops-arrow" v-click>→</div>
  <div class="gitops-step" v-click>
    <div class="gitops-icon">⚙️</div>
    <div class="gitops-label">Operator provisions<br/>on Harvester</div>
  </div>
  <div class="gitops-arrow" v-click>→</div>
  <div class="gitops-step" v-click>
    <div class="gitops-icon">✅</div>
    <div class="gitops-label">Cluster appears<br/>in Rancher</div>
  </div>
</div>

<div v-click class="gitops-note">
  Platform team reviews a PR. No ticket system. No manual provisioning. No dual kubectl contexts.
</div>

<style>
.gitops-flow {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 1rem;
  margin: 3rem 0 2rem;
}
.gitops-step {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.6rem;
  background: #f0f0f0;
  border-radius: 8px;
  padding: 1.2rem 1rem;
  min-width: 130px;
  text-align: center;
}
.gitops-icon { font-size: 2rem; }
.gitops-label { font-size: 0.8rem; line-height: 1.4; }
.gitops-label code { font-size: 0.75rem; background: #e0e0e0; padding: 0.1rem 0.3rem; border-radius: 3px; }
.gitops-arrow { font-size: 1.8rem; color: var(--slidev-theme-primary); font-weight: 700; }
.gitops-note {
  text-align: center;
  font-style: italic;
  font-size: 0.9rem;
  color: #555;
  border-top: 1px solid #ddd;
  padding-top: 1rem;
}
</style>

---
layout: cover
image: /examples/write_sticker_1.jpg
---

# Live Demo

---
layout: default
---

# What We'll See

<div class="demo-stack">
  <div class="demo-layer demo-git">
    <strong>Git</strong> — WorkloadCluster manifests in <code>fleet/clusters/demo/</code>
  </div>
  <div class="demo-arrow">↓ Fleet syncs</div>
  <div class="demo-layer demo-op">
    <strong>Operator</strong> — reconciles WorkloadCluster CRs on the management cluster
  </div>
  <div class="demo-arrow">↓ provisions</div>
  <div class="demo-layer demo-clusters">
    <div class="demo-cluster demo-shared"><strong>k3k-shared</strong><br/><span>virtual cluster</span></div>
    <div class="demo-cluster demo-hcp"><strong>k3k-hcp</strong><br/><span>hosted CP + VM</span></div>
    <div class="demo-cluster demo-rke2"><strong>rke2-vm</strong><br/><span>dedicated cluster</span></div>
  </div>
  <div class="demo-arrow">↓ running on</div>
  <div class="demo-layer demo-harvester">
    <strong>Harvester</strong> — VMs, OVN overlay networks, Longhorn storage
  </div>
</div>

<style>
.demo-stack {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.4rem;
  margin-top: 1rem;
}
.demo-layer {
  width: 75%;
  padding: 0.6rem 1.2rem;
  border-radius: 6px;
  font-size: 0.85rem;
  text-align: center;
}
.demo-layer code { background: rgba(0,0,0,0.08); padding: 0.1rem 0.3rem; border-radius: 3px; font-size: 0.8rem; }
.demo-git       { background: #e8f4f8; }
.demo-op        { background: #d0eaf4; }
.demo-clusters  { display: flex; gap: 1rem; width: 75%; justify-content: center; }
.demo-cluster   { padding: 0.6rem 1rem; border-radius: 6px; text-align: center; font-size: 0.8rem; flex: 1; }
.demo-cluster span { font-size: 0.7rem; color: #666; }
.demo-shared    { background: #d0eaf4; }
.demo-hcp       { background: #a8d5e8; }
.demo-rke2      { background: #7bbfd8; color: #fff; }
.demo-rke2 span { color: #e0f0f8; }
.demo-harvester { background: #333; color: #fff; }
.demo-arrow     { font-size: 0.85rem; color: #999; }
</style>

---
layout: section-simple
---

# Lessons Learned

---
layout: default
---

# There Is No "Best" Model

<table class="tradeoff-table">
  <thead><tr><th>Model</th><th>Isolation</th><th>Cost</th><th>Density</th><th>Best For</th></tr></thead>
  <tbody>
    <tr class="row-vc"><td>k3k-shared (virtual cluster)</td><td>★★☆☆☆</td><td>$</td><td>★★★★★</td><td>Dev, CI, short-lived</td></tr>
    <tr class="row-hcp"><td>k3k-hcp (hosted CP + VMs)</td><td>★★★★☆</td><td>$$$</td><td>★★★☆☆</td><td>Teams, SaaS tenants</td></tr>
    <tr class="row-ded"><td>rke2-vm (dedicated)</td><td>★★★★★</td><td>$$$$</td><td>★★☆☆☆</td><td>Regulated, GPU, prod</td></tr>
  </tbody>
</table>

*One operator, one CRD, one Git workflow — for all three.*

<style>
.tradeoff-table { width: 100%; border-collapse: collapse; font-size: 0.9rem; margin-bottom: 0.75rem; }
.tradeoff-table th { padding: 0.5rem 0.75rem; text-align: left; border-bottom: 2px solid #ccc; font-weight: 700; }
.tradeoff-table td { padding: 0.5rem 0.75rem; }
.row-vc  { background: #d0eaf4; }
.row-hcp { background: #a8d5e8; }
.row-ded { background: #7bbfd8; }
</style>

---
layout: default
---

# Takeaways

- **HCI gives you the primitives** — VMs, networking, storage as Kubernetes resources; no proprietary cloud APIs
- **Treat clusters as resources**, not products — a CRD and an operator replace your provisioning runbook
- **GitOps is your self-service layer** — developers open PRs, platform teams review; no ticket system, no waiting
- **The isolation model is a runtime decision** — same YAML shape, different `type` field, different infrastructure
- **Rancher + Fleet + Harvester + k3k = a coherent private cloud stack** — all open source, all GitOps-native

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
        <span>manuel.zapf@codecentric.cloud</span>
      </div>
    </div>
  </div>
  
</div>
