# OCI image for open-etc-pool, built with Nix — no Docker daemon required:
#
#   nix build .#dockerImage && docker load < result
#
# This is only a convenience for people who deploy with Docker. The native
# way to consume the flake is packages.default / nixosModules.default; nothing
# here is needed for a Nix/NixOS deployment.
{ pkgs, package }:
pkgs.dockerTools.buildLayeredImage {
  name = "open-etc-pool";
  tag = "latest";
  contents = [
    package
    pkgs.cacert
  ];
  config = {
    Entrypoint = [ "/bin/open-etc-pool" ];
    Cmd = [ "/config.json" ];
    ExposedPorts = {
      "8888/tcp" = { }; # HTTP getwork
      "8008/tcp" = { }; # stratum
      "8080/tcp" = { }; # stats API
    };
  };
}
