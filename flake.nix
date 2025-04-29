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
              natt = final.buildGoModule {
                pname = "natt";
                version = "0.0.1";
                src = final.lib.sourceByRegex ./. [
                  "go.mod"
                  "go.sum"
                  "cmd"
                  "cmd/.*"
                  "internal"
                  "internal/.*"
                ];
                subPackages = [ "cmd/nattc" "cmd/natts" ];
                vendorHash = "sha256-Yk2gx1c/HcwQ6TgT1DX+sUgKIrcugG6gD9QWTlFZlwM=";
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
          default = pkgs.natt;
          arm64 = pkgs.pkgsCross.aarch64-multiplatform.natt;
        };
        devShells.default = shell;
      }
    );
}
