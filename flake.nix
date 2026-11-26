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
      version = "0.7.0";
    in
    {
      devShells = each (
        pkgs:
        let
          # nixpkgs membawa wails 2.17.0, sedangkan go.mod memakai library 2.17.1.
          # CLI 2.17.0 men-downgrade go.mod tiap `wails dev`; pin CLI ke versi library.
          wails = pkgs.wails.overrideAttrs (_old: {
            version = "2.17.1";
            src = pkgs.fetchFromGitHub {
              owner = "wailsapp";
              repo = "wails";
              tag = "v2.17.1";
              hash = "sha256-QVYEAnIMKb7aW/u2tUdmKIAge0vYdctiVVEKZqE0+KM=";
            };
            vendorHash = "sha256-jvj8M8bXcxyOkoxfnYaTuG7Zk7+O6f1mdnpws2AHvto=";
          });

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

      packages = each (pkgs: {
        cleanlie = pkgs.callPackage ./package.nix {
          inherit version;
          commit = self.shortRev or "dirty";
          source = ./.;
        };
        default = self.packages.${pkgs.stdenv.hostPlatform.system}.cleanlie;
      });

      # Overlay supaya bisa dipakai dari konfigurasi Nix lain:
      #   nixpkgs.overlays = [ inputs.cleanlie.overlays.default ];
      #   environment.systemPackages = [ pkgs.cleanlie ];
      overlays.default = final: prev: {
        cleanlie = final.callPackage ./package.nix {
          inherit version;
          commit = self.shortRev or "dirty";
          source = ./.;
        };
      };

      # Jalur pintas untuk NixOS:
      #   imports = [ inputs.cleanlie.nixosModules.default ];
      nixosModules.default =
        { pkgs, ... }:
        {
          nixpkgs.overlays = [ self.overlays.default ];
          environment.systemPackages = [ pkgs.cleanlie ];
        };

      apps = each (pkgs: {
        default = {
          type = "app";
          program = "${self.packages.${pkgs.stdenv.hostPlatform.system}.cleanlie}/bin/cleanlie";
        };
      });
    };
}
