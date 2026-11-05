{
  description = "agy-notebooks-mcp: Gemini Notebooks / NotebookLM Read-Only MCP Server for Google Antigravity (agy)";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "agy-notebooks-mcp";
          version = "0.1.0";
          src = ./.;

          vendorHash = null; # pure standard library Go

          subPackages = [ "cmd/agy-notebooks-mcp" ];

          meta = with pkgs.lib; {
            description = "Gemini Notebooks / NotebookLM Read-Only MCP Server for Google Antigravity (agy)";
            homepage = "https://github.com/andre-amorim/agy-notebooks-mcp";
            license = licenses.gpl3Only;
            mainProgram = "agy-notebooks-mcp";
          };
        };

        apps.default = flake-utils.lib.mkApp {
          drv = self.packages.${system}.default;
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            golangci-lint
          ];

          shellHook = ''
            export CGO_ENABLED=0
          '';
        };
      });
}
