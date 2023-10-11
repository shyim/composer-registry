{ buildGoModule, version ? "dev" }:

buildGoModule rec {
    pname = "composer-registry";
    inherit version;
    src = ../.;
    vendorSha256 = "sha256-sFjwj4UoC/5xqwH6uHxRI0AsorOVVicWVTtNhwP/Xxs=";
}