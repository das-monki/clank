{
  description = "Clank - minimal task/project management";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "clank";
          version = "0.1.0";
          src = ./.;
          vendorHash = "sha256-UK94jG3Iguc4zJ6KACGEbfombNjsJ+zl1xtZwrvL33g=";

          ldflags = [ "-s" "-w" ];

          meta = {
            description = "Minimal task/project management";
            mainProgram = "clank";
          };
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            gotools
            sqlite
          ];
        };
      }
    ) // {
      nixosModules.default = import ./module.nix self;
    };
}
