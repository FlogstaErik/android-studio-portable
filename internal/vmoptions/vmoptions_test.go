package vmoptions

import (
	"os"
	"path/filepath"
	"testing"
)

type paths struct{ bundled, user, defaults string }

func setup(t *testing.T) paths {
	t.Helper()
	dir := t.TempDir()
	return paths{
		bundled:  filepath.Join(dir, "studio64.exe.vmoptions"),
		user:     filepath.Join(dir, "studio.vmoptions"),
		defaults: filepath.Join(dir, "studio.vmoptions.default"),
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func sync(t *testing.T, p paths, want Result) {
	t.Helper()
	got, ok, err := Sync(p.bundled, p.user, p.defaults)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if !ok {
		t.Fatal("Sync reported user file unusable")
	}
	if got != want {
		t.Fatalf("Sync result = %d, want %d", got, want)
	}
}

func TestFirstRunCreatesUserFile(t *testing.T) {
	p := setup(t)
	write(t, p.bundled, "-Xmx2g\n")
	sync(t, p, Created)
	if got := read(t, p.user); got != "-Xmx2g\n" {
		t.Errorf("user = %q", got)
	}
	if got := read(t, p.defaults); got != "-Xmx2g\n" {
		t.Errorf("defaults = %q", got)
	}
}

func TestNoUpgradeKeepsEdits(t *testing.T) {
	p := setup(t)
	write(t, p.bundled, "-Xmx2g\n")
	sync(t, p, Created)
	write(t, p.user, "-Xmx8g\n")
	sync(t, p, Unchanged)
	if got := read(t, p.user); got != "-Xmx8g\n" {
		t.Errorf("user = %q", got)
	}
}

func TestUpgradeRefreshesUneditedFile(t *testing.T) {
	p := setup(t)
	write(t, p.bundled, "-Xmx2g\n")
	sync(t, p, Created)
	write(t, p.bundled, "-Xmx4g\n-XX:+UseG1GC\n")
	sync(t, p, Refreshed)
	if got := read(t, p.user); got != "-Xmx4g\n-XX:+UseG1GC\n" {
		t.Errorf("user = %q", got)
	}
}

func TestUpgradeKeepsEditedFile(t *testing.T) {
	p := setup(t)
	write(t, p.bundled, "-Xmx2g\n")
	sync(t, p, Created)
	write(t, p.user, "-Xmx8g\n")
	write(t, p.bundled, "-Xmx4g\n")
	sync(t, p, KeptEdited)
	if got := read(t, p.user); got != "-Xmx8g\n" {
		t.Errorf("user = %q", got)
	}
	if got := read(t, p.defaults); got != "-Xmx4g\n" {
		t.Errorf("defaults = %q", got)
	}
	// A later launch on the same version leaves everything alone.
	sync(t, p, Unchanged)
}

func TestExistingInstallWithoutSnapshot(t *testing.T) {
	p := setup(t)
	write(t, p.bundled, "-Xmx2g\n")
	write(t, p.user, "-Xmx8g\n")
	sync(t, p, Snapshotted)
	if got := read(t, p.user); got != "-Xmx8g\n" {
		t.Errorf("user = %q", got)
	}
}

func TestBundledNeverWritten(t *testing.T) {
	p := setup(t)
	write(t, p.bundled, "-Xmx2g\n")
	sync(t, p, Created)
	write(t, p.user, "-Xmx8g\n")
	sync(t, p, Unchanged)
	if got := read(t, p.bundled); got != "-Xmx2g\n" {
		t.Errorf("bundled = %q", got)
	}
}

func TestMissingBundled(t *testing.T) {
	p := setup(t)
	if _, ok, err := Sync(p.bundled, p.user, p.defaults); err == nil || ok {
		t.Fatalf("ok = %v, err = %v; want unusable with error", ok, err)
	}
	write(t, p.user, "-Xmx8g\n")
	if _, ok, err := Sync(p.bundled, p.user, p.defaults); err == nil || !ok {
		t.Fatalf("ok = %v, err = %v; want usable with error", ok, err)
	}
}
