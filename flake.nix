{
  description = "Cleanlie — disk cleaner GUI for NixOS";

  inputs.nixpkgs.url = "nixpkgs";

  outputs =
    { self, nixpkgs }:
    let
      # Paket, overlay, dan nixosModules hanya masuk akal di Linux: build-nya
      # butuh gtk3/webkitgtk_4_1, dan runtime-nya butuh nix/systemd/polkit.
      linuxSystems = [
        "x86_64-linux"
        "aarch64-linux"
      ];
      # macOS cuma dapat devShell untuk `go test` dan kerja di sisi Go+frontend.
      # x86_64-darwin sengaja tidak didaftarkan: nixpkgs 26.11 sudah drop platform itu.
      darwinSystems = [ "aarch64-darwin" ];
      eachLinux = f: nixpkgs.lib.genAttrs linuxSystems (system: f nixpkgs.legacyPackages.${system});
      each = f: nixpkgs.lib.genAttrs (linuxSystems ++ darwinSystems) (system: f nixpkgs.legacyPackages.${system});
      version = "0.8.0";

      devShellFor =
        pkgs:
        let
          # nixpkgs membawa wails 2.17.0, sedangkan go.mod memakai library 2.17.1.
          # CLI 2.17.0 men-downgrade go.mod tiap `wails dev`; pin CLI ke versi library.
          # Override ini platform-agnostik; nixpkgs sendiri yang membatasi gtk/webkit ke Linux.
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

          commonPackages = with pkgs; [
            go
            gopls
            gotools
            pkg-config
            gnumake
            git
            jq
            nodejs # frontend/test/ui-smoke.mjs (jsdom)
          ];

          # Library runtime untuk `wails dev` dan binary cgo hasil `go build`.
          runtimeLibs = with pkgs; [
            gtk3
            glib
            webkitgtk_4_1
            libsoup_3
          ];
        in
        if pkgs.stdenv.hostPlatform.isLinux then
          pkgs.mkShell {
            packages = commonPackages ++ [ wails ] ++ runtimeLibs;

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
          }
        else
          pkgs.mkShell {
            # Tanpa gtk/webkitgtk: backend Wails di macOS memakai WKWebView sistem.
            packages = commonPackages ++ [ wails ];

            env = {
              CGO_ENABLED = "1";
              GOTOOLCHAIN = "local";
              # Wails (lewat go-toast) memakai UTType yang butuh framework ini;
              # tanpa flag eksplisit, linker gagal dengan "_OBJC_CLASS_$_UTType".
              CGO_LDFLAGS = "-framework UniformTypeIdentifiers";
            };

            shellHook = ''
              echo "cleanlie dev shell (macOS) · go $(go version | awk '{print $3}') · wails $(wails version 2>/dev/null | head -1)"
              echo "  make test    # go test ./..."
              echo "  make build   # bin/cleanlie"
              echo "  make app     # Cleanlie.app (wails build)"
              echo "  catatan: target cleanlie (NixOS) hanya bisa di-build di Linux; di macOS fitur Nix tetap jalan bila Nix terpasang."
            '';
          };
    in
    {
      devShells = each (pkgs: {
        default = devShellFor pkgs;
      });

      packages = eachLinux (pkgs: {
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

      apps = eachLinux (pkgs: {
        default = {
          type = "app";
          program = "${self.packages.${pkgs.stdenv.hostPlatform.system}.cleanlie}/bin/cleanlie";
        };
      });
    };
}
