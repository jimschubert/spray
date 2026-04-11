package plug

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func writeStubScript(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	assert.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755))
	return path
}

func TestLookupDirs(t *testing.T) {
	tests := []struct {
		name     string
		pathDirs []string
	}{
		{
			name:     "home dir is first when PATH is empty",
			pathDirs: []string{},
		},
		{
			name:     "path entries follow home dir in order",
			pathDirs: []string{"/a", "/b", "/c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// can't use t.Parallel() here because of t.Setenv.
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("PATH", strings.Join(tt.pathDirs, string(filepath.ListSeparator)))

			dirs := LookupDirs()

			assert.Equal(t, filepath.Join(home, ".spray", "plugins"), dirs[0])
			assert.Equal(t, len(tt.pathDirs)+1, len(dirs))
			for i, p := range tt.pathDirs {
				assert.Equal(t, p, dirs[i+1])
			}
		})
	}
}

func TestFindIn(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T) (dirs []string, wantPath string)
		wantErr     bool
		errContains string
	}{
		{
			name: "returns path when executable exists in dir",
			setup: func(t *testing.T) ([]string, string) {
				dir := t.TempDir()
				want := writeStubScript(t, dir, "spray-emitter-myplugin")
				return []string{dir}, want
			},
		},
		{
			name: "prefers first dir when binary exists in multiple",
			setup: func(t *testing.T) ([]string, string) {
				dir1, dir2 := t.TempDir(), t.TempDir()
				want := writeStubScript(t, dir1, "spray-emitter-myplugin")
				writeStubScript(t, dir2, "spray-emitter-myplugin")
				return []string{dir1, dir2}, want
			},
		},
		{
			name: "skips entries that are directories named like the binary",
			setup: func(t *testing.T) ([]string, string) {
				dir := t.TempDir()
				assert.NoError(t, os.Mkdir(filepath.Join(dir, "spray-emitter-myplugin"), 0o755))
				return []string{dir}, ""
			},
			wantErr: true,
		},
		{
			name: "error message includes emitter name and binary name",
			setup: func(t *testing.T) ([]string, string) {
				return []string{t.TempDir()}, ""
			},
			wantErr:     true,
			errContains: "spray-emitter-myplugin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dirs, wantPath := tt.setup(t)
			path, err := findIn("myplugin", dirs)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, wantPath, path)
			}
		})
	}
}

func TestFind(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T)
		emitterName string
		wantErr     bool
		errContains string
	}{
		{
			name: "finds plugin in ~/.spray/plugins",
			setup: func(t *testing.T) {
				home := t.TempDir()
				t.Setenv("HOME", home)
				t.Setenv("PATH", "")
				pluginsDir := filepath.Join(home, ".spray", "plugins")
				assert.NoError(t, os.MkdirAll(pluginsDir, 0o755))
				writeStubScript(t, pluginsDir, "spray-emitter-myplugin")
			},
			emitterName: "myplugin",
		},
		{
			name: "error message includes emitter and binary name",
			setup: func(t *testing.T) {
				t.Setenv("HOME", t.TempDir())
				t.Setenv("PATH", "")
			},
			emitterName: "notexist",
			wantErr:     true,
			errContains: "spray-emitter-notexist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// can't use t.Parallel() here because of t.Setenv.
			tt.setup(t)
			path, err := Find(tt.emitterName)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Zero(t, path)
			} else {
				assert.NoError(t, err)
				assert.NotZero(t, path)
			}
		})
	}
}
