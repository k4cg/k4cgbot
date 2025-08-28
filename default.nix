let pkgs = import <nixpkgs> { };
in pkgs.buildGoModule rec {
  pname = "k4cgbot";
  version = "1.3.0";
  src = pkgs.lib.cleanSource ./.;
  vendorHash = "sha256-g90As9KIgCqlMB+8OByEeDMF2YwZy5Mz4vqKbdPuRXo=";
}
