{
  description = "Find OpenSpec projects on your local machine" ;

  inputs.nixpkgs.url = "nixpkgs/nixos-25.05";

  outputs = { self, nixpkgs }:
    let
      supportedSystems = [ "x86_64-linux" "x86_64-darwin" "aarch64-linux" "aarch64-darwin" ];
      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;
      nixpkgsFor = forAllSystems (system: import nixpkgs { inherit system; });
      version = builtins.replaceStrings ["\n"] [""] (builtins.readFile ./src/VERSION);
    in
    {
      nixosModules.default = import ./module.nix self;

      packages = forAllSystems (system:
        let
          pkgs = nixpkgsFor.${system};
        in
        {
          specgetty = pkgs.callPackage ./package.nix { inherit version; };
        });

      defaultPackage = forAllSystems (system: self.packages.${system}.specgetty);

      devShells = forAllSystems (system:
        let
          pkgs = nixpkgsFor.${system};
        in
        {
          default = pkgs.mkShell {
            buildInputs = with pkgs; [
            ];
          };
        });
    };
}
