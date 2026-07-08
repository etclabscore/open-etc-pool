# NixOS module for open-etc-pool. Consumed via the flake:
#
#   imports = [ open-etc-pool.nixosModules.default ];
#   services.open-etc-pool = {
#     enable = true;
#     settings = { coin = "etc"; proxy.enabled = true; /* ... */ };
#     # or, to keep secrets out of the Nix store:
#     # configFile = "/run/secrets/open-etc-pool.json";
#   };
{ self }:
{ config, lib, pkgs, ... }:
let
  cfg = config.services.open-etc-pool;
  jsonFormat = pkgs.formats.json { };
  configFile =
    if cfg.configFile != null then
      cfg.configFile
    else
      jsonFormat.generate "open-etc-pool.json" cfg.settings;
in
{
  options.services.open-etc-pool = {
    enable = lib.mkEnableOption "the open-etc-pool Ethereum Classic mining pool";

    package = lib.mkOption {
      type = lib.types.package;
      default = self.packages.${pkgs.stdenv.hostPlatform.system}.default;
      defaultText = lib.literalExpression "open-etc-pool.packages.\${system}.default";
      description = "The open-etc-pool package to run.";
    };

    settings = lib.mkOption {
      type = jsonFormat.type;
      default = { };
      example = lib.literalExpression ''{ coin = "etc"; proxy.enabled = true; }'';
      description = ''
        Pool configuration, serialized to config.json (see config.example.json).
        NOTE: this is world-readable in the Nix store — use `configFile` for
        anything sensitive (e.g. the Redis password).
      '';
    };

    configFile = lib.mkOption {
      type = lib.types.nullOr lib.types.path;
      default = null;
      description = ''
        Path to a config.json managed outside Nix. When set, `settings` is
        ignored. Use this to keep secrets out of the world-readable Nix store.
      '';
    };

    user = lib.mkOption {
      type = lib.types.str;
      default = "open-etc-pool";
      description = "User the service runs as.";
    };

    group = lib.mkOption {
      type = lib.types.str;
      default = "open-etc-pool";
      description = "Group the service runs as.";
    };
  };

  config = lib.mkIf cfg.enable {
    users.users = lib.mkIf (cfg.user == "open-etc-pool") {
      open-etc-pool = {
        isSystemUser = true;
        group = cfg.group;
      };
    };
    users.groups = lib.mkIf (cfg.group == "open-etc-pool") {
      open-etc-pool = { };
    };

    systemd.services.open-etc-pool = {
      description = "open-etc-pool — Ethereum Classic mining pool";
      wantedBy = [ "multi-user.target" ];
      after = [ "network-online.target" ];
      wants = [ "network-online.target" ];
      serviceConfig = {
        ExecStart = "${lib.getExe cfg.package} ${configFile}";
        User = cfg.user;
        Group = cfg.group;
        Restart = "on-failure";
        RestartSec = 5;
        # Hardening — the pool keeps its state in Redis, no filesystem writes.
        NoNewPrivileges = true;
        ProtectSystem = "strict";
        ProtectHome = true;
        PrivateTmp = true;
        PrivateDevices = true;
        ProtectControlGroups = true;
        ProtectKernelModules = true;
        ProtectKernelTunables = true;
        RestrictSUIDSGID = true;
      };
    };
  };
}
