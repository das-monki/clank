self: { config, lib, pkgs, ... }:

let
  cfg = config.services.clank;
in
{
  options.services.clank = {
    enable = lib.mkEnableOption "clank task manager";

    port = lib.mkOption {
      type = lib.types.port;
      default = 8080;
      description = "Port to listen on";
    };

    dataDir = lib.mkOption {
      type = lib.types.path;
      default = "/var/lib/clank";
      description = "Directory to store the SQLite database";
    };
  };

  config = lib.mkIf cfg.enable {
    systemd.services.clank = {
      description = "Clank Task Manager";
      wantedBy = [ "multi-user.target" ];
      after = [ "network.target" ];

      serviceConfig = {
        ExecStart = "${self.packages.${pkgs.system}.default}/bin/clank serve --port ${toString cfg.port} --db ${cfg.dataDir}/clank.db";
        StateDirectory = "clank";
        DynamicUser = true;
        Restart = "on-failure";
        RestartSec = "5s";
      };
    };
  };
}
