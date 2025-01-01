let pkgs = import <nixpkgs> { };
in pkgs.buildGoModule rec {
  pname = "k4cgbot";
  version = "1.1.0";
  src = pkgs.lib.cleanSource ./.;
  vendorHash = "sha256-i+Z43KrxYAJ39L88yTpH7KX3/y+r7o/R54evOoDYTiI=";
}