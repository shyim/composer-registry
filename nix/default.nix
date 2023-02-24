{ buildGoModule, version ? "dev" }:

buildGoModule rec {
    pname = "composer-registry";
    inherit version;
    src = ../.;
    vendorSha256 = "sha256-cCToHmwFHR04DzKlpAQIFXmHV74Su6NF/U9jJLeZgo0=";
}