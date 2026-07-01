# From Clusters to Control Planes: Rethinking Multi-Tenancy at Scale

Demo repository for the talk "From Clusters to Control Planes: Rethinking Multi-Tenancy at Scale".

Slides: https://slides.manuelzapf.io/from-clusters-to-controlplanes

## What's in here

| Directory | What it is |
|-----------|------------|
| `demo/` | Go TUI that drives the live demo — provisions and tears down CAPI clusters |
| `slides/` | Slidev presentation source |
| `clusters/` | CAPI cluster manifests (k3s+KubeVirt, Kamaji+KubeVirt, k3k) |
| `fleet/` | Fleet bundles for bootstrapping the management cluster (KubeVirt, Kamaji, k3k, CAPI providers, Gateway API) |
| `gitrepos/` | Fleet GitRepo pointing at this repo |

## Prerequisites

- [Nix](https://nixos.org/download) with flakes enabled
- A k3s management cluster running Rancher + Turtles + KubeVirt
- `kubectl` context `ranchero-k3s` pointing at the management cluster

## Demo TUI

The TUI provisions three downstream clusters via CAPI and tracks their status:

```sh
nix run .#demo
```

Or with Make:

```sh
make demo           # run TUI
make clusters-apply # apply cluster manifests manually
make clusters-reset # delete + re-apply all clusters
```

The TUI watches milestone timestamps (control plane ready, workers ready, Rancher-imported) derived from CAPI conditions.

## Management Cluster Bootstrap

The management cluster runs k3s with Rancher, Turtles, KubeVirt, Kamaji, k3k, and all CAPI providers installed via Fleet.

```sh
make kubeconfig     # pull kubeconfig from the server
make fleet-apply    # apply Fleet GitRepo (points at this repo)
```

Fleet then installs everything from the `fleet/` directory.

## Slides

The presentation is built with [Slidev](https://sli.dev) and deployed at
`https://slides.manuelzapf.io/from-clusters-to-controlplanes`.

### Local dev

```sh
nix develop .#slides
npm run dev         # live preview at localhost:3030
```

### Deploy

Requires `skopeo` and `kubectl` in PATH (or `nix run .#deploy-slides` which provides all tools):

```sh
nix run .#deploy-slides
```

This:
1. Generates D2 diagrams (`slides/diagrams/*.d2` → `public/diagrams/*.svg`)
2. Generates the QR code SVG
3. Builds the Slidev bundle with the correct base path
4. Builds a Docker image via `nix build --impure .#slides-image` (no Docker daemon needed)
5. Pushes to `docker.io/santode/slides:latest` via skopeo
6. Rolls out the deployment on k3s

> **Note:** Slidev is pinned to `52.15.2`. Version `52.16.0` introduced a routing bug
> (doubled base path on navigation) that is fixed upstream in
> [PR #2630](https://github.com/slidevjs/slidev/pull/2630) but not yet released.

## Nix flake outputs

The flake is structured in four blocks:

**`packages`** — things `nix build` produces as store artifacts:
- `demo` / `default` — compiled Go binary (via gomod2nix, fully sandboxed)
- `slides-image` — amd64 Docker image tarball (nginx + slides dist); requires `SLIDES_DIST_PATH` and `--impure` because `dist/` is not git-tracked

**`apps`** — things `nix run` executes:
- `demo` / `default` — runs the compiled TUI binary
- `deploy-slides` — shell script with all tools pinned on PATH (node, d2, python, skopeo); renders diagrams, generates QR code, runs `npm build`, calls `nix build --impure .#slides-image`, pushes via skopeo, rolls out on k3s

**`devShells`** — things `nix develop` drops you into:
- `default` — Go dev environment (go, gopls, kubectl, helm, k9s, node)
- `slides` — slides dev environment (node, d2, python+qrcode)

`pkgs` vs `pkgsLinux`: host-arch packages are used for the build tooling; `pkgsLinux` (hardcoded x86_64-linux) is used for nginx and fakeNss inside the container so the image is always amd64 regardless of build host.

```
nix run .#demo              # run the demo TUI
nix run .#deploy-slides     # build + deploy slides
nix build .#slides-image    # build slides Docker image (needs --impure + SLIDES_DIST_PATH)
nix develop                 # Go dev shell
nix develop .#slides        # Node/D2/Python dev shell for slides
```
