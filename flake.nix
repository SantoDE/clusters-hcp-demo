{
  description = "HCP demo — cluster provisioning TUI and dev shell";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    gomod2nix = {
      url = "github:nix-community/gomod2nix";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.flake-utils.follows = "flake-utils";
    };
  };

  outputs = { self, nixpkgs, flake-utils, gomod2nix }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        # x86_64-linux binaries for the k3s cluster — fetched from nixpkgs cache, not built locally
        pkgsLinux = nixpkgs.legacyPackages.x86_64-linux;
        inherit (gomod2nix.legacyPackages.${system}) buildGoApplication;

        # dist/ is a build artifact, not git-tracked. Pass via SLIDES_DIST_PATH env var
        # and build with: SLIDES_DIST_PATH=$(realpath slides/dist) nix build --impure .#slides-image
        slidesDist = let path = builtins.getEnv "SLIDES_DIST_PATH"; in
          if path != ""
          then builtins.path { path = /. + path; name = "slides-dist"; }
          else builtins.throw "Set SLIDES_DIST_PATH to slides/dist and use --impure";

        # Runs on host arch — just file copying, no architecture-specific binaries
        slidesContent = pkgs.runCommand "slides-content" {} ''
          mkdir -p $out/usr/share/nginx/html/from-clusters-to-controlplanes
          cp -r ${slidesDist}/. $out/usr/share/nginx/html/from-clusters-to-controlplanes/
        '';

        slidesNginxConf = pkgs.writeTextDir "etc/nginx/nginx.conf" ''
          user nobody;
          worker_processes auto;
          pid /tmp/nginx.pid;

          events { worker_connections 1024; }

          http {
              include ${pkgsLinux.nginx}/conf/mime.types;
              default_type application/octet-stream;
              access_log /dev/stdout;
              error_log /dev/stderr warn;

              server {
                  listen 80;
                  server_name _;

                  location /from-clusters-to-controlplanes/ {
                      root /usr/share/nginx/html;
                      try_files $uri $uri/ /from-clusters-to-controlplanes/index.html;
                      location ~* \.html$ {
                          add_header Cache-Control "no-store, no-cache, must-revalidate";
                      }
                  }

                  location = / {
                      return 301 /from-clusters-to-controlplanes/;
                  }
              }
          }
        '';

        slidesImage = pkgs.dockerTools.buildLayeredImage {
          name = "santode/slides";
          tag = "latest";
          architecture = "amd64";
          contents = [
            pkgsLinux.fakeNss
            pkgsLinux.nginx
            slidesNginxConf
            slidesContent
          ];
          config = {
            Cmd = [ "${pkgsLinux.nginx}/bin/nginx" "-g" "daemon off;" ];
            ExposedPorts."80/tcp" = {};
          };
        };

      in {
        packages.demo = buildGoApplication {
          pname = "hcp-demo";
          version = "0.1.0";
          src = ./demo;
          modules = ./demo/gomod2nix.toml;
        };

        packages.slides-image = slidesImage;

        packages.default = self.packages.${system}.demo;

        apps.demo = {
          type = "app";
          program = "${self.packages.${system}.demo}/bin/demo";
        };

        apps.default = self.apps.${system}.demo;

        apps.deploy-slides = {
          type = "app";
          program = "${pkgs.writeShellApplication {
            name = "deploy-slides";
            runtimeInputs = with pkgs; [
              nodejs_22
              d2
              (python3.withPackages (ps: [ ps.qrcode ]))
              skopeo
            ];
            text = ''
              repo=$(git rev-parse --show-toplevel)
              cd "$repo/slides"

              for f in diagrams/*.d2; do
                d2 "$f" "public/diagrams/$(basename "''${f%.d2}").svg"
              done

              python3 - <<'PYEOF'
              import qrcode, qrcode.image.svg
              qr = qrcode.QRCode(error_correction=qrcode.constants.ERROR_CORRECT_L, box_size=10, border=2)
              qr.add_data('https://slides.manuelzapf.io/from-clusters-to-controlplanes')
              qr.make(fit=True)
              qr.make_image(image_factory=qrcode.image.svg.SvgPathImage).save('public/qr-slides.svg')
              PYEOF

              npm run build -- --base /from-clusters-to-controlplanes/

              image=$(SLIDES_DIST_PATH="$PWD/dist" nix build --impure "$repo#slides-image" --no-link --print-out-paths)
              skopeo copy --insecure-policy "docker-archive:$image" "docker://docker.io/santode/slides:latest"

              kubectl apply -f k8s/deployment.yaml
              kubectl rollout restart deployment/slides -n slides
            '';
          }}/bin/deploy-slides";
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            gotools
            gomod2nix.packages.${system}.default
            kubectl
            kubernetes-helm
            clusterctl
            k9s
            nodejs_22
          ];
        };

        devShells.slides = pkgs.mkShell {
          buildInputs = with pkgs; [
            nodejs_22
            d2
            (python3.withPackages (ps: with ps; [ qrcode ]))
          ];
        };
      });
}
