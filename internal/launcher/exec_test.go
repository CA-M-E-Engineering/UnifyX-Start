package launcher

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	if ready := os.Getenv("UNIFYX_TEST_HELPER_READY"); ready != "" {
		if err := os.WriteFile(ready, []byte("ready"), 0600); err != nil {
			os.Exit(2)
		}
		time.Sleep(time.Hour)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestBrowserCompatibility(t *testing.T) {
	for version, want := range map[string]bool{
		"Chromium 111.0.5563.0":                   true,
		"Chromium 148.0.7778.0 built on Debian":   true,
		"Google Chrome for Testing 153.0.8010.12": true,
		"Chromium 110.0.0.0":                      false,
		"Google Chrome 156.0.8078.47":             false,
		"Microsoft Edge 111.0.1661.41":            true,
		"Microsoft Edge 156.0.4314.42":            true,
		"Microsoft Edge 110.0.1587.69":            false,
		"Microsoft Edge WebView2 156.0.4314.42":   false,
		"":                                        false,
	} {
		t.Run(version, func(t *testing.T) {
			if err := validateBrowserVersion(version, false); (err == nil) != want {
				t.Fatalf("compatible=%v, want=%v; error=%v", err == nil, want, err)
			}
		})
	}
}

func TestNativeFullscreenBrowserCompatibility(t *testing.T) {
	for version, want := range map[string]bool{
		"Chromium 88.0.4324.0":          true,
		"Chromium 91.0.4472.164 stable": true,
		"Chromium 110.0.0.0":            true,
		"Chromium 87.0.0.0":             false,
		"Microsoft Edge 110.0.1587.69":  true,
		"Google Chrome 156.0.8078.47":   false,
	} {
		t.Run(version, func(t *testing.T) {
			if err := validateBrowserVersion(version, true); (err == nil) != want {
				t.Fatalf("compatible=%v, want=%v; error=%v", err == nil, want, err)
			}
		})
	}
}

func TestNativeFullscreenKeepsTouchGuardWithoutMainWorldShim(t *testing.T) {
	dir, err := setupTempExtension(true)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	data, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Minimum string `json:"minimum_chrome_version"`
		Scripts []struct {
			JS    []string `json:"js"`
			CSS   []string `json:"css"`
			World string   `json:"world"`
		} `json:"content_scripts"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Minimum != "88" || len(manifest.Scripts) != 1 {
		t.Fatalf("unexpected manifest: %s", data)
	}
	script := manifest.Scripts[0]
	if script.World != "" || !slices.Equal(script.JS, []string{"content.js"}) || !slices.Equal(script.CSS, []string{"content.css"}) {
		t.Fatalf("input protection was changed: %s", data)
	}
	args := buildChromeArgs("http://127.0.0.1:8080/", "profile", dir)
	if !slices.Contains(args, "--kiosk") || !slices.Contains(args, "--start-fullscreen") {
		t.Fatal("native fullscreen flags are missing")
	}
}

func TestEdgeDiscoveryLocations(t *testing.T) {
	root := t.TempDir()
	t.Setenv("ProgramFiles", root)
	for goos, want := range map[string]string{
		"windows": filepath.Join(root, "Microsoft", "Edge", "Application", "msedge.exe"),
		"darwin":  filepath.Join("/Applications", "Microsoft Edge.app", "Contents", "MacOS", "Microsoft Edge"),
		"linux":   "microsoft-edge-stable",
	} {
		if !slices.Contains(browserCandidates(goos), want) {
			t.Errorf("%s discovery is missing %s", goos, want)
		}
	}
}

func TestExplicitBrowserPathDoesNotFallBack(t *testing.T) {
	if _, err := findChromeBinary(context.Background(), filepath.Join(t.TempDir(), "missing-browser"), false); err == nil {
		t.Fatal("invalid explicit browser path was ignored")
	}
}

func TestBrowserDiscoveryCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := findChromeBinary(ctx, "", false); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}

func TestInstalledBrowserSetup(t *testing.T) {
	path := os.Getenv("UNIFYX_TEST_BROWSER")
	if path == "" {
		t.Skip("set UNIFYX_TEST_BROWSER for the installed-browser check")
	}
	app, err := New(context.Background(), path, t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	if _, err := os.Stat(filepath.Join(app.extension, "manifest.json")); err != nil {
		t.Fatal(err)
	}
	app.Close()
	if _, err := os.Stat(app.extension); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("extension was not cleaned up: %v", err)
	}
}

func TestExtensionAssets(t *testing.T) {
	dir, err := setupTempExtension(false)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	data, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Scripts []struct {
			JS    []string `json:"js"`
			CSS   []string `json:"css"`
			World string   `json:"world"`
		} `json:"content_scripts"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	mainWorld := false
	for _, script := range manifest.Scripts {
		for _, name := range append(script.JS, script.CSS...) {
			if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
				t.Fatal(err)
			}
			if name == "fullscreen.js" {
				mainWorld = script.World == "MAIN"
			}
		}
	}
	if !mainWorld {
		t.Fatal("fullscreen hook must execute in the page's MAIN world")
	}
}

func TestCleanupProfilePreservesOtherPreferences(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "Default"), 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "Default", "Preferences")
	data := []byte(`{"profile": {"exit_type": "Crashed", "exited_cleanly":false, "other":42}, "counter":9007199254740993}`)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"SingletonLock", "SingletonSocket", "SingletonCookie"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := cleanupStaleLocks(dir); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var prefs struct {
		Profile struct {
			ExitType string `json:"exit_type"`
			Clean    bool   `json:"exited_cleanly"`
			Other    int
		}
		Counter json.Number
	}
	if err := json.Unmarshal(got, &prefs); err != nil {
		t.Fatal(err)
	}
	if prefs.Profile.ExitType != "Normal" || !prefs.Profile.Clean || prefs.Profile.Other != 42 || prefs.Counter != "9007199254740993" {
		t.Fatalf("unexpected preferences: %s", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "SingletonLock")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("lock remains: %v", err)
	}
}

func TestMalformedPreferencesAreNotOverwritten(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "Default"), 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "Default", "Preferences")
	if err := os.WriteFile(path, []byte("{broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := cleanupStaleLocks(dir); err == nil {
		t.Fatal("expected invalid JSON error")
	}
	data, _ := os.ReadFile(path)
	if string(data) != "{broken" {
		t.Fatalf("preferences overwritten: %s", data)
	}
}

func TestRunReturnsStartFailure(t *testing.T) {
	dir := t.TempDir()
	app := &App{browser: filepath.Join(dir, "missing"), profile: dir, extension: dir}
	if err := app.Run(context.Background(), "about:blank"); err == nil {
		t.Fatal("expected startup failure")
	}
}

func TestRunCancellationStopsChildProcess(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	ready := filepath.Join(dir, "ready")
	t.Setenv("UNIFYX_TEST_HELPER_READY", ready)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	app := &App{browser: self, profile: dir, extension: dir}
	done := make(chan error, 1)
	go func() { done <- app.Run(ctx, "about:blank") }()
	for {
		if _, err := os.Stat(ready); err == nil {
			break
		}
		select {
		case err := <-done:
			t.Fatalf("helper stopped before ready: %v", err)
		case <-ctx.Done():
			t.Fatal("helper never started")
		case <-time.After(10 * time.Millisecond):
		}
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected cancellation, got %v", err)
		}
	case <-time.After(7 * time.Second):
		t.Fatal("browser process did not stop")
	}
}
