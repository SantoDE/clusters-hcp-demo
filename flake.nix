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
        inherit (gomod2nix.legacyPackages.${system}) buildGoApplication;
      in {
        packages.demo = buildGoApplication {
          pname = "hcp-demo";
          version = "0.1.0";
          src = ./demo;
          modules = ./demo/gomod2nix.toml;
        };

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
              chmod -R a+r dist/

              docker build --platform linux/amd64 -t docker.io/santode/slides:latest .
              docker push docker.io/santode/slides:latest

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
