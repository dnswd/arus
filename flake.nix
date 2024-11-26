{
  description = "A Nix-flake-based Go 1.22 development environment";

  inputs.nixpkgs.url = "https://flakehub.com/f/NixOS/nixpkgs/0.1.*.tar.gz";
  inputs.flake-parts.url = "github:hercules-ci/flake-parts";
  inputs.systems.url = "github:nix-systems/default";
  inputs.process-compose-flake.url = "github:Platonic-Systems/process-compose-flake";
  inputs.services-flake.url = "github:juspay/services-flake";

  outputs = { ... }@inputs:
    let
      goVersion = 22; # Change this to update the whole stack
      dbName = "sample";
    in
    inputs.flake-parts.lib.mkFlake { inherit inputs; } rec {
      systems = import inputs.systems;

      flake = {
        overlays.default =
          final: prev: {
            go = final."go_1_${toString goVersion}";
          };
      };

      imports = [
        inputs.process-compose-flake.flakeModule
      ];

      perSystem = { pkgs, config, ... }: {
        # `process-compose.foo` will add a flake package output called "foo".
        # Therefore, this will add a default package that you can build using
        # `nix build` and run using `nix run`.
        process-compose."default" = { config, ... }:
          {
            imports = [
              inputs.services-flake.processComposeModules.default
            ];

            services.postgres."pg1" = {
              enable = true;
              package = pkgs.postgresql_17.withPackages (pkgs: [ pkgs.pg_uuidv7 ]);
              initialDatabases = [{ name = dbName; }];
            };

            settings.processes.pgweb =
              let
                pgcfg = config.services.postgres.pg1;
              in
              {
                environment.PGWEB_DATABASE_URL = pgcfg.connectionURI { inherit dbName; };
                command = pkgs.pgweb;
                depends_on."pg1".condition = "process_healthy";
              };

            settings.processes.test = {
              command = pkgs.writeShellApplication {
                name = "pg1-test";
                runtimeInputs = [ config.services.postgres.pg1.package ];
                text = # sh
                  ''
                    echo 'SELECT version();' | psql -h 127.0.0.1 ${dbName}
                  '';
              };
              depends_on."pg1".condition = "process_healthy";
            };
          };

        devShells.default = pkgs.mkShell {
          inputsFrom = [
            config.process-compose."default".services.outputs.devShell
          ];
          nativeBuildInputs = [ pkgs.just ];
          packages = with pkgs; [
            go

            # goimports, godoc, etc.
            gotools

            # https://github.com/golangci/golangci-lint
            golangci-lint

            # live reload
            air

            # db schema management
            atlas

            # inspection tools
            wireshark


          ];
          buildInputs = [
            (pkgs.writeShellApplication {
              name = "migrate";
              text = # sh
                ''
                  #!${pkgs.runtimeShell}
                  echo -e "Checking migrations"
                  atlas migrate validate
                  echo -e "Applying migrations"
                  atlas migrate hash
                  atlas migrate apply -u postgresql://127.0.0.1:5432/${dbName}\?sslmode=disable
                  echo -e "Done, creating schema snapshot at schema.sql"
                  pg_dump --schema-only postgresql://127.0.0.1:5432/${dbName}\?sslmode=disable > schema.sql
                '';
            })
          ];
          shellHook = ''
            GOROOT="$(dirname $(dirname $(which go)))/share/go"
            BUILD_REGISTRY="ghcr.io"
            export GOROOT
            export DOCKER_HOST
            export BUILD_REGISTRY
            unset GOPATH;
          '';
        };
      };
    };


}
