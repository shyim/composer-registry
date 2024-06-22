{ buildGoModule, version ? "dev" }:

buildGoModule rec {
    pname = "composer-registry";
    inherit version;
    src = ../.;
    vendorHash = "sha256-gbkE04rc6FJoxLqHVmyw+84QdJkSYoXMAINGFtWeg9w=";
}
