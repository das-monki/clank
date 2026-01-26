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

        clank = pkgs.buildGoModule {
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

        # Helper to create a wrapped clank CLI with pre-configured API URL
        mkClankCli = { apiUrl }: pkgs.symlinkJoin {
          name = "clank-cli";
          paths = [ clank ];
          buildInputs = [ pkgs.makeWrapper ];
          postBuild = ''
            wrapProgram $out/bin/clank \
              --set CLANK_API "${apiUrl}"
          '';
          meta = {
            description = "Clank CLI configured for ${apiUrl}";
            mainProgram = "clank";
          };
        };
      in
      {
        packages.default = clank;

        # Function to create a wrapped CLI with pre-configured API URL
        lib.mkCli = mkClankCli;

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
