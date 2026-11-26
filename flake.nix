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
      version = "0.6.0";
    in
    {
      devShells = each (
        pkgs:
        let
          # Library runtime untuk `wails dev` dan binary cgo hasil `go build`.
          runtimeLibs = with pkgs; [
            gtk3
            glib
            webkitgtk_4_1
            libsoup_3
          ];
        in
        {
          default = pkgs.mkShell {
            packages = with pkgs; [
              go
              gopls
              gotools
              wails
              pkg-config
              gnumake
              git
              jq
              nodejs # frontend/test/ui-smoke.mjs (jsdom)
            ] ++ runtimeLibs;

            env = {
              CGO_ENABLED = "1";
              # Pakai go dari shell ini; jangan unduh toolchain lain diam-diam.
              GOTOOLCHAIN = "local";
              # WebKitGTK di Wayland/Hyprland kadang butuh ini; aman untuk dev.
              WEBKIT_DISABLE_COMPOSITING_MODE = "1";
              LD_LIBRARY_PATH = pkgs.lib.makeLibraryPath runtimeLibs;
            };

            shellHook = ''
              echo "cleanlie dev shell · go $(go version | awk '{print $3}') · wails $(wails version 2>/dev/null | head -1)"
              echo "  make dev     # wails dev (hot reload)"
              echo "  make build   # bin/cleanlie"
              echo "  make test"
            '';
          };
        }
      );

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
              gappsWrapperArgs+=(--set-default WEBKIT_DISABLE_COMPOSITING_MODE 1)
            '';

            postInstall = ''
              install -Dm644 ${./packaging/cleanlie.desktop} $out/share/applications/cleanlie.desktop
              install -Dm644 ${./packaging/cleanlie.svg} $out/share/icons/hicolor/scalable/apps/cleanlie.svg
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
