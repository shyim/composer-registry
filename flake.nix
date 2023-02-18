{
  description = "Composer Registry";

  # Nixpkgs / NixOS version to use.
  inputs.nixpkgs.url = "nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let

      # to work with older version of flakes
      lastModifiedDate = self.lastModifiedDate or self.lastModified or "19700101";

      # Generate a user-friendly version number.
      version = builtins.substring 0 8 lastModifiedDate;

      # System types to support.
      supportedSystems = [ "x86_64-linux" "x86_64-darwin" "aarch64-linux" "aarch64-darwin" ];

      # Helper function to generate an attrset '{ x86_64-linux = f "x86_64-linux"; ... }'.
      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;

      # Nixpkgs instantiated for supported system types.
      nixpkgsFor = forAllSystems (system: import nixpkgs { inherit system; });
    in
    {

      # Provide some binary packages for selected system types.
      packages = forAllSystems (system:
        let
          pkgs = nixpkgsFor.${system};
        in
        rec {
          composer-registry = pkgs.buildGoModule {
            pname = "composer-registry";
            inherit version;
            src = ./.;

            nativeBuildInputs = [ pkgs.installShellFiles ];

            vendorSha256 = "sha256-VoJZY0IcdlMJ3g64DZatCbP68GvWH8sFvL5o7diYLQM=";

            postInstall = ''
              export HOME="$(mktemp -d)"
              installShellCompletion --cmd composer-registry \
                --bash <($out/bin/composer-registry completion bash) \
                --zsh <($out/bin/composer-registry completion zsh) \
                --fish <($out/bin/composer-registry completion fish)
            '';

          };
          default = composer-registry;
        });

      apps = forAllSystems (system: rec {
        composer-registry = {
          type = "app";
          program = "${self.packages.${system}.composer-registry}/bin/composer-registry";
        };
        default = composer-registry;
      });

      defaultPackage = forAllSystems (system: self.packages.${system}.default);

      defaultApp = forAllSystems (system: self.apps.${system}.default);

      devShell = forAllSystems (system:
        let pkgs = nixpkgsFor.${system};
        in pkgs.mkShell {
          buildInputs = with pkgs; [ go_1_20 golangci-lint ];
        });

      formatter = forAllSystems (
        system:
        nixpkgs.legacyPackages.${system}.nixpkgs-fmt
      );
    };
}
