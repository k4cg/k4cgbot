let pkgs = import <nixpkgs> { };
in pkgs.buildGoModule rec {
  pname = "k4cgbot";
  version = "0.0.1";
  src = pkgs.lib.cleanSource ./.;
  vendorHash = "sha256-rhizQadk+DfsD/iqW61/xzTQSZn5VnMERtkgHtCeJns=";
}