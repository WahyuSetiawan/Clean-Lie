{
  description = "Cleanlie — disk cleaner GUI for NixOS";

  inputs.nixpkgs.url = "nixpkgs";

  outputs =
    { self, nixpkgs }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
      ];
      each = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
      version = "0.5.0";
    in
    {
      devShells = each (pkgs: {
        default = pkgs.mkShell {
          packages = with pkgs; [
            go
            wails
            pkg-config
            gtk3
            webkitgtk_4_1
            glib
            jq
          ];

          env = {
            CGO_ENABLED = "1";
            # WebKitGTK di Wayland/Hyprland kadang butuh ini; aman untuk dev.
            WEBKIT_DISABLE_COMPOSITING_MODE = "1";
          };

          shellHook = ''
            echo "cleanlie dev shell · go $(go version | awk '{print $3}') · wails $(wails version 2>/dev/null | head -1)"
            echo "  make dev     # wails dev"
            echo "  make build   # bin/cleanlie"
            echo "  make test"
          '';
        };
      });

      packages = each (
        pkgs:
        let
          runtimeDeps = with pkgs; [
            nix
            systemd
            polkit
            glib # gio trash
            xdg-utils # xdg-open
            gtk3
            webkitgtk_4_1
          ];
        in
        {
          cleanlie = pkgs.buildGoModule {
            pname = "cleanlie";
            inherit version;
            src = ./.;
            vendorHash = "sha256-y+hhovS1wVZkSXig8LOpjlek20O9VErVYmw+VGLfIfI=";

            nativeBuildInputs = [
              pkgs.pkg-config
              pkgs.wrapGAppsHook3
            ];
            buildInputs = [
              pkgs.gtk3
              pkgs.webkitgtk_4_1
              pkgs.glib
            ];

            env.CGO_ENABLED = "1";
            tags = [
              "webkit2_41"
              "desktop"
              "production"
            ];
            ldflags = [
              "-s"
              "-w"
              "-X"
              "main.version=${version}"
              "-X"
              "main.commit=${self.shortRev or "dirty"}"
            ];

            # PATH untuk aksi root & perintah sistem; pkexec membuang environment sesi.
            preFixup = ''
              gappsWrapperArgs+=(--prefix PATH : ${pkgs.lib.makeBinPath runtimeDeps})
            '';

            meta = {
              description = "Disk cleaner and analyzer for NixOS";
              homepage = "https://github.com/Juragan-Koding/cleanlie";
              mainProgram = "cleanlie";
              platforms = pkgs.lib.platforms.linux;
            };
          };

          default = self.packages.${pkgs.system}.cleanlie;
        }
      );

      apps = each (pkgs: {
        default = {
          type = "app";
          program = "${self.packages.${pkgs.system}.cleanlie}/bin/cleanlie";
        };
      });
    };
}
