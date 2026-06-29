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
          ];
        };
      });
}
