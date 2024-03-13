{ buildGoModule, version ? "dev" }:

buildGoModule rec {
    pname = "composer-registry";
    inherit version;
    src = ../.;
    vendorSha256 = "sha256-nzh7+5GErnXu9BayhyLZh89jRrbxAD1oqLe5v+uP4uQ=";
}