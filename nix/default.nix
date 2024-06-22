{ buildGoModule, version ? "dev" }:

buildGoModule rec {
    pname = "composer-registry";
    inherit version;
    src = ../.;
    vendorHash = "sha256-8ZQT6EZ8MSeg1tTTQn9/0kICnsabghmYNAy9WVdrMHE=";
}
