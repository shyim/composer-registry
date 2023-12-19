{ buildGoModule, version ? "dev" }:

buildGoModule rec {
    pname = "composer-registry";
    inherit version;
    src = ../.;
    vendorSha256 = "sha256-6A7RlhdnqZz/vLLF0JIxy9Z6b4OZA5Z7QG0KJ659ioQ=";
}