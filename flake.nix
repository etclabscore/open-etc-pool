{
  description = "open-etc-pool — Ethereum Classic mining pool";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.05";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    { self, nixpkgs, flake-utils }:
    # System-independent outputs — how you consume the flake on NixOS.
    {
      # imports = [ open-etc-pool.nixosModules.default ];
      nixosModules.default = import ./nix/module.nix { inherit self; };

      # nixpkgs.overlays = [ open-etc-pool.overlays.default ];  ->  pkgs.open-etc-pool
      overlays.default = final: _prev: {
        open-etc-pool = self.packages.${final.stdenv.hostPlatform.system}.default;
      };
    }
    // flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs { inherit system; };

        open-etc-pool = pkgs.buildGoModule {
          pname = "open-etc-pool";
          version = "0.0.0+${self.shortRev or "dirty"}";
          src = self;

          # Hash of the module dependencies. Recompute after go.mod/go.sum
          # changes: set to pkgs.lib.fakeHash, run `nix build .#default`, and
          # copy the expected hash from the error.
          vendorHash = "sha256-PRJY75NLDADFkz+VxVO2v4t4Q3cDN3k9QaRcKoC5UyY=";

          subPackages = [ "." ];

          # The pool is pure Go — build a static binary (smaller image, no libc).
          env.CGO_ENABLED = "0";
          ldflags = [ "-s" "-w" ];

          meta = with pkgs.lib; {
            description = "Open source Ethereum Classic mining pool";
            homepage = "https://github.com/etclabscore/open-etc-pool";
            license = licenses.gpl3Only;
            mainProgram = "open-etc-pool";
          };
        };
      in
      {
        packages =
          {
            default = open-etc-pool;
            open-etc-pool = open-etc-pool;
          }
          # Optional OCI image for Docker users — defined in ./nix/docker.nix
          # so no Docker concepts leak into the flake. Linux-only (dockerTools).
          // pkgs.lib.optionalAttrs pkgs.stdenv.isLinux {
            dockerImage = import ./nix/docker.nix {
              inherit pkgs;
              package = open-etc-pool;
            };
          };

        devShells.default = pkgs.mkShell {
          packages = [
            pkgs.go
            pkgs.gopls
            pkgs.gotools
            pkgs.redis # for `go test ./storage/...`
            pkgs.nodejs # for the frontend
          ];
        };

        # Validate the NixOS module by evaluating it in a real system config.
        checks = pkgs.lib.optionalAttrs pkgs.stdenv.isLinux {
          nixos-module =
            let
              machine = nixpkgs.lib.nixosSystem {
                inherit system;
                modules = [
                  self.nixosModules.default
                  {
                    boot.loader.grub.enable = false;
                    fileSystems."/" = {
                      device = "/dev/null";
                      fsType = "ext4";
                    };
                    system.stateVersion = "25.05";
                    services.open-etc-pool = {
                      enable = true;
                      settings = {
                        coin = "etc";
                        proxy.enabled = true;
                      };
                    };
                  }
                ];
              };
            in
            pkgs.runCommandLocal "nixos-module-eval" { } ''
              # Forces evaluation of the pool's systemd unit; fails if the
              # module is broken.
              printf '%s' ${
                pkgs.lib.escapeShellArg machine.config.systemd.services.open-etc-pool.serviceConfig.ExecStart
              } > "$out"
            '';
        };
      }
    );
}
