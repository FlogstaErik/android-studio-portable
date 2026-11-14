// Package vmoptions keeps the user's Android Studio VM options file in the
// portable data folder in step with the defaults shipped by Android Studio.
package vmoptions

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
)

// Result describes what Sync did.
type Result int

const (
	// Unchanged means the user file was already up to date.
	Unchanged Result = iota
	// Created means the user file did not exist and was created from the bundled defaults.
	Created
	// Refreshed means the bundled defaults changed (Android Studio was upgraded)
	// and the user file, which had not been edited, was replaced with them.
	Refreshed
	// KeptEdited means the bundled defaults changed but the user file had been
	// edited, so it was kept as it is.
	KeptEdited
	// Snapshotted means no record of the previous defaults existed, so the
	// current bundled defaults were recorded and the user file was kept.
	Snapshotted
	// BundledMissing means the bundled file could not be read and the existing
	// user file was kept.
	BundledMissing
)

// Sync makes sure userPath exists and follows the defaults in bundledPath.
//
// defaultsPath holds a copy of the bundled defaults as they were the last time
// Sync ran. When the bundled file changes, an unedited user file is replaced
// with the new defaults, and an edited one is left alone. The bundled file is
// never written to.
//
// The returned bool reports whether userPath exists and can be used.
func Sync(bundledPath, userPath, defaultsPath string) (Result, bool, error) {
	bundled, err := os.ReadFile(bundledPath)
	if err != nil {
		if fileExists(userPath) {
			return BundledMissing, true, fmt.Errorf("cannot read bundled VM options: %w", err)
		}
		return BundledMissing, false, fmt.Errorf("cannot read bundled VM options: %w", err)
	}

	user, err := os.ReadFile(userPath)
	if errors.Is(err, fs.ErrNotExist) {
		if err := os.WriteFile(userPath, bundled, 0o644); err != nil {
			return Created, false, fmt.Errorf("cannot create %s: %w", userPath, err)
		}
		return Created, true, writeDefaults(defaultsPath, bundled)
	} else if err != nil {
		return Unchanged, false, fmt.Errorf("cannot read %s: %w", userPath, err)
	}

	previous, err := os.ReadFile(defaultsPath)
	if errors.Is(err, fs.ErrNotExist) {
		return Snapshotted, true, writeDefaults(defaultsPath, bundled)
	} else if err != nil {
		return Unchanged, true, fmt.Errorf("cannot read %s: %w", defaultsPath, err)
	}

	if bytes.Equal(previous, bundled) {
		return Unchanged, true, nil
	}

	result := KeptEdited
	if bytes.Equal(user, previous) {
		if err := os.WriteFile(userPath, bundled, 0o644); err != nil {
			return Unchanged, true, fmt.Errorf("cannot refresh %s: %w", userPath, err)
		}
		result = Refreshed
	}
	return result, true, writeDefaults(defaultsPath, bundled)
}

func writeDefaults(path string, content []byte) error {
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("cannot write %s: %w", path, err)
	}
	return nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
