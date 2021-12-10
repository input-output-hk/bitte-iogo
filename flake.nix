{
  description = "Flake for the Bitte iogo command";

  inputs = {
    devshell.url = "github:numtide/devshell";
    inclusive.url = "github:input-output-hk/nix-inclusive";
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    utils.url = "github:kreisys/flake-utils";
  };

  outputs = { self, nixpkgs, utils, devshell, ... }@inputs:
    utils.lib.simpleFlake {
      systems = [ "x86_64-linux" ];
      inherit nixpkgs;

      preOverlays = [ devshell.overlay ];

      overlay = final: prev: {
        iogo = prev.buildGoModule rec {
          pname = "iogo";
          version = "2021.12.10.001";
          vendorSha256 = "sha256-g36jy/TBBvW7M1Wsdj5NxXQyotBsw2t6L2RnvBICCaU=";

          src = inputs.inclusive.lib.inclusive ./. [
            ./cue.go
            ./fixtures
            ./go.mod
            ./go.sum
            ./job.hcl
            ./json2hcl.go
            ./json2hcl_test.go
            ./login.go
            ./main.go
          ];

          ldflags = [
            "-s"
            "-w"
            "-X main.buildVersion=${version}"
            "-X main.buildCommit=${self.rev or "dirty"}"
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

      devShell = { devshell, go, goimports, gopls, gocode, gcc }:
        devshell.mkShell {
          name = "bitte-iogo-shell";
          packages = [ go goimports gopls gocode gcc ];
        };
    };
}
