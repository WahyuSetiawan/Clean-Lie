{
  lib,
  buildGoModule,
  pkg-config,
  wrapGAppsHook3,
  gtk3,
  webkitgtk_4_1,
  glib,
  nix,
  systemd,
  polkit,
  xdg-utils,
  version ? "0.7.0",
  commit ? "unknown",
  source ? ./.,
}:

buildGoModule {
  pname = "cleanlie";
  inherit version;
  src = source;
  vendorHash = "sha256-y+hhovS1wVZkSXig8LOpjlek20O9VErVYmw+VGLfIfI=";

  nativeBuildInputs = [
    pkg-config
    wrapGAppsHook3
  ];
  buildInputs = [
    gtk3
    webkitgtk_4_1
    glib
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
    "main.commit=${commit}"
  ];

  # PATH untuk aksi root & perintah sistem; pkexec membuang environment sesi.
  preFixup = ''
    gappsWrapperArgs+=(--prefix PATH : ${
      lib.makeBinPath [
        nix
        systemd
        polkit
        glib
        xdg-utils
      ]
    })
    gappsWrapperArgs+=(--set-default WEBKIT_DISABLE_COMPOSITING_MODE 1)
  '';

  postInstall = ''
    install -Dm644 ${source}/packaging/cleanlie.desktop $out/share/applications/cleanlie.desktop
    install -Dm644 ${source}/packaging/cleanlie.svg $out/share/icons/hicolor/scalable/apps/cleanlie.svg
  '';

  meta = {
    description = "Disk cleaner and analyzer for NixOS";
    homepage = "https://github.com/WahyuSetiawan/Clean-Lie";
    mainProgram = "cleanlie";
    platforms = lib.platforms.linux;
  };
}
