{
  description = "Flake for the Bitte iogo command";

  inputs = {
    devshell.url = "github:numtide/devshell";
    inclusive.url = "github:input-output-hk/nix-inclusive";
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-21.05";
    utils.url = "github:kreisys/flake-utils";
  };

  outputs = { self, nixpkgs, utils, devshell, ... }@inputs:
    utils.lib.simpleFlake {
      systems = [ "x86_64-linux" ];
      inherit nixpkgs;

      preOverlays = [ devshell.overlay ];

      overlay = final: prev: {
        iogo = prev.buildGoModule {
          pname = "iogo";
          version = "1.0.0";
          vendorSha256 = "sha256-XWbyybBYlQCWhSwDTrFtmL4xFS6bxHl7wwGf7f+9pjE=";

          src = inputs.inclusive.lib.inclusive ./. [
            ./cue.go
            ./go.mod
            ./go.sum
            ./json2hcl.go
            ./job.hcl
            ./main.go
            ./login.go
          ];

          postInstall = ''
            mv $out/bin/bitte-iogo $out/bin/iogo
          '';
        };
      };

      packages = { iogo }@pkgs: {
        inherit iogo;
        defaultPackage = iogo;
      };

      hydraJobs = { iogo }@pkgs: pkgs;

      devShell = { devshell, go, goimports, gopls, gocode }:
        devshell.mkShell {
          name = "bitte-iogo-shell";
          packages = [ go goimports gopls gocode ];
        };
    };
}
