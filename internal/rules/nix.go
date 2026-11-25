package rules

import "strings"

// NixStore adalah root store Nix. Store tidak pernah disentuh lewat trash;
// pembersihannya hanya lewat `nix-collect-garbage` / `nix store delete`.
const NixStore = "/nix/store"

// IsNixStorePath melaporkan apakah path berada di dalam /nix/store.
func IsNixStorePath(p string) bool {
	return p == NixStore || strings.HasPrefix(p, NixStore+"/")
}

// NixProfilesPath mengembalikan direktori profile per user.
func NixProfilesPath(home string) string {
	return home + "/.local/state/nix/profiles"
}
