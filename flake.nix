{
  description = "A basic gomod2nix flake";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  inputs.flake-utils.url = "github:numtide/flake-utils";
  inputs.gomod2nix.url = "github:nix-community/gomod2nix";

  outputs = { self, nixpkgs, flake-utils, gomod2nix }:
    (flake-utils.lib.eachDefaultSystem
      (system:
        let
          pkgs = import nixpkgs {
            inherit system;
            overlays = [ gomod2nix.overlays.default ];
          };
          goEnv = pkgs.mkGoEnv { pwd = ./.; };
        in
        {
          packages.default = pkgs.buildGoApplication {
            pname = "smutje";
            version = "1.0";
            pwd = ./.;
            src = ./.;
            # subPackages = [ ./cmds/smutje ./cmds/smd-fmt ];
            modules = ./gomod2nix.toml;
          };

          devShells.default = pkgs.mkShell {
            packages = [
              goEnv
              pkgs.gomod2nix
            ];
          };
        })
    );
}
