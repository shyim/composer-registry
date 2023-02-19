{ buildGoModule, version ? "dev" }:

buildGoModule rec {
    pname = "composer-registry";
    inherit version;
    src = ../.;
    vendorSha256 = "sha256-jctlMpXVRBwDgfDPVeibV9hOywMboTxGNd4mxdffzWY=";
}