package plug

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// LookupDirs returns the ordered list of directories searched for plugin executables.
// Plugins are executables named `spray-emitter-<name>`.
func LookupDirs() []string {
	var dirs []string

	// ~/.spray/plugins
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".spray", "plugins"))
	}

	// then, PATH
	for _, p := range filepath.SplitList(os.Getenv("PATH")) {
		dirs = append(dirs, p)
	}

	return dirs
}

// Find returns the absolute path of the plugin executable for the given emitter name, or an error.
func Find(name string) (string, error) {
	return findIn(name, LookupDirs())
}

// findIn searches dirs for a spray-emitter-<name> executable, then falls back to exec.LookPath.
// Used for testing with custom directories.
func findIn(name string, dirs []string) (string, error) {
	bin := "spray-emitter-" + name
	for _, dir := range dirs {
		candidate := filepath.Join(dir, bin)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}

	if path, err := exec.LookPath(bin); err == nil {
		return path, nil
	}
	return "", fmt.Errorf(
		"no plugin found for emitter %q (looked for %q in %s)",
		name, bin, strings.Join(dirs, ", "),
	)
}
