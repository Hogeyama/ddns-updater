{
  description = "Update DNDS record";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils/main";
  };

  outputs = { nixpkgs, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
          overlays = [
            (final: _: {
              ddns-updater = final.buildGoModule {
                pname = "ddns-updater";
                version = "0.0.1";
                src = final.lib.sourceByRegex ./. [
                  "go.mod"
                  "go.sum"
                  "cmd"
                  "cmd/.*"
                  "internal"
                  "internal/.*"
                ];
                subPackages = [ "cmd/ddns-updater" ];
                vendorHash = "sha256-90UlMI7XslN/f3pK3RY0rOywS7Ws4IbCMmCdXBP1tdQ=";
                proxyVendor = true;
                env.CGO_ENABLED = 0;
              };
            })
          ];
        };
        shell = pkgs.mkShell {
          packages = [
            pkgs.go
            pkgs.gopls
            pkgs.gotools
          ];
          shellHook = ''
            export GOMODCACHE=$PWD/.gomodcache
            mkdir -p $GOMODCACHE
          '';
        };
      in
      {
        packages = {
          default = pkgs.ddns-updater;
          arm64 = pkgs.pkgsCross.aarch64-multiplatform.ddns-updater;
        };
        devShells.default = shell;
      }
    );
}
