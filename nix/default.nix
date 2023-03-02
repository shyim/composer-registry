{ buildGoModule, version ? "dev" }:

buildGoModule rec {
    pname = "composer-registry";
    inherit version;
    src = ../.;
    vendorSha256 = "sha256-biTc49tgo2yKa0ceJ8+whnB4h1ywjoSuqCj5SLvzj7M=";
}