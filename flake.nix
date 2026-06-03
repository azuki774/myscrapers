{
  description = "Development shell for myscrapers";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-26.05";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { nixpkgs, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
      in {
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            bashInteractive
            coreutils
            direnv
            go
            gopls
            nodejs_24
          ];

          shellHook = ''
            export GOPATH="$PWD/.gopath"
            export GOMODCACHE="$PWD/.gopath/pkg/mod"
            export PLAYWRIGHT_BROWSERS_PATH="$PWD/myscraper/.playwright"
          '';
        };
      });
}
