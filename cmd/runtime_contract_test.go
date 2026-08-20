package cmd

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const (
	mockDockerProcessEnv = "KUBEORCH_TEST_DOCKER_PROCESS"
	testExecutablePerm   = 0750
)

func TestMain(m *testing.M) {
	if os.Getenv(mockDockerProcessEnv) == "1" {
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func useMockDocker(t *testing.T) {
	t.Helper()
	mockDir := t.TempDir()
	mockPath := filepath.Join(mockDir, "docker")
	if runtime.GOOS == "windows" {
		mockPath += ".exe"
	}
	testExecutable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	executableData, err := os.ReadFile(testExecutable)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mockPath, executableData, testExecutablePerm); err != nil {
		t.Fatal(err)
	}

	t.Setenv(mockDockerProcessEnv, "1")
	t.Setenv("PATH", mockDir)
}

func TestProjectMarkerDiscoveryFromNestedDirectory(t *testing.T) {
	projectPath := t.TempDir()
	uiPath := filepath.Join(projectPath, "ui")
	corePath := filepath.Join(projectPath, "core")
	for _, path := range []string{uiPath, corePath, filepath.Join(uiPath, "app", "dashboard")} {
		if err := os.MkdirAll(path, dirPerm); err != nil {
			t.Fatal(err)
		}
	}

	project := &ProjectConfig{
		Path:     projectPath,
		UIPath:   uiPath,
		CorePath: corePath,
		Mode:     "development",
	}
	if err := writeProjectMarker(project); err != nil {
		t.Fatal(err)
	}

	loaded, err := loadProjectMarker(filepath.Join(uiPath, "app", "dashboard"))
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Path != projectPath || loaded.UIPath != uiPath || loaded.CorePath != corePath {
		t.Fatalf("unexpected project config: %#v", loaded)
	}

	data, err := os.ReadFile(filepath.Join(projectPath, projectMarkerDir, projectMarkerFilename))
	if err != nil {
		t.Fatal(err)
	}
	var marker projectMarker
	if err := json.Unmarshal(data, &marker); err != nil {
		t.Fatal(err)
	}
	if marker.UIPath != "ui" || marker.CorePath != "core" {
		t.Fatalf("expected portable paths, got UI=%q Core=%q", marker.UIPath, marker.CorePath)
	}
}

func TestProjectMarkerErrorsAreActionable(t *testing.T) {
	t.Run("invalid JSON", func(t *testing.T) {
		projectPath := t.TempDir()
		markerDir := filepath.Join(projectPath, projectMarkerDir)
		if err := os.MkdirAll(markerDir, dirPerm); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(markerDir, projectMarkerFilename), []byte("{"), configFilePerm); err != nil {
			t.Fatal(err)
		}
		_, err := loadProjectMarker(projectPath)
		if err == nil || !strings.Contains(err.Error(), "invalid project marker") || !strings.Contains(err.Error(), "orchcli init") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("unsupported version", func(t *testing.T) {
		projectPath := t.TempDir()
		markerDir := filepath.Join(projectPath, projectMarkerDir)
		if err := os.MkdirAll(markerDir, dirPerm); err != nil {
			t.Fatal(err)
		}
		data := []byte(`{"version":99,"mode":"production"}`)
		if err := os.WriteFile(filepath.Join(markerDir, projectMarkerFilename), data, configFilePerm); err != nil {
			t.Fatal(err)
		}
		_, err := loadProjectMarker(projectPath)
		if err == nil || !strings.Contains(err.Error(), "unsupported project marker version 99") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestAtomicWritePreservesTargetBeforeRenameFailure(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), projectMarkerFilename)
	original := []byte(`{"version":1,"mode":"production"}`)
	if err := os.WriteFile(targetPath, original, configFilePerm); err != nil {
		t.Fatal(err)
	}

	expectedErr := errors.New("simulated interruption")
	err := writeFileAtomicallyWithHook(
		targetPath,
		[]byte(`{"version":1,"ui_path":"ui","mode":"ui-dev"}`),
		configFilePerm,
		func(string) error { return expectedErr },
	)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected simulated interruption, got %v", err)
	}

	actual, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(actual) != string(original) {
		t.Fatalf("target changed after failed replacement: %q", actual)
	}

	tempFiles, err := filepath.Glob(filepath.Join(filepath.Dir(targetPath), ".project.json-*.tmp"))
	if err != nil {
		t.Fatal(err)
	}
	if len(tempFiles) != 0 {
		t.Fatalf("temporary files were not cleaned up: %v", tempFiles)
	}
}

func TestExistingCheckoutPrerequisitesDoNotRequireGit(t *testing.T) {
	useMockDocker(t)
	previousAutoInstall := autoInstall
	autoInstall = false
	defer func() { autoInstall = previousAutoInstall }()

	if err := checkPrerequisites(false); err != nil {
		t.Fatalf("existing-checkout prerequisites unexpectedly required Git: %v", err)
	}
}

func TestFlaglessInitPreservesDevelopmentSourcePaths(t *testing.T) {
	projectPath := t.TempDir()
	uiPath := filepath.Join(projectPath, "ui")
	corePath := filepath.Join(projectPath, "core")
	for _, path := range []string{uiPath, corePath} {
		if err := os.MkdirAll(path, dirPerm); err != nil {
			t.Fatal(err)
		}
	}
	if err := setProjectConfig(projectPath, uiPath, corePath); err != nil {
		t.Fatal(err)
	}
	useMockDocker(t)
	previousDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if chdirErr := os.Chdir(projectPath); chdirErr != nil {
		t.Fatal(chdirErr)
	}
	t.Cleanup(func() { _ = os.Chdir(previousDir) })
	if setupErr := setupProduction(); setupErr != nil {
		t.Fatal(setupErr)
	}

	loaded, err := loadProjectMarker(projectPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Mode != "development" || loaded.UIPath != uiPath || loaded.CorePath != corePath {
		t.Fatalf("flagless init changed the development marker: %#v", loaded)
	}
}

func TestResolveExistingCheckout(t *testing.T) {
	projectPath := t.TempDir()
	uiPath := filepath.Join(projectPath, "ui")
	if err := os.MkdirAll(uiPath, dirPerm); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(uiPath, "package.json"), []byte("{}"), composeFilePerm); err != nil {
		t.Fatal(err)
	}

	resolved, err := resolveExistingCheckout(projectPath, "ui", "UI", "package.json")
	if err != nil {
		t.Fatal(err)
	}
	if resolved != uiPath {
		t.Fatalf("expected %s, got %s", uiPath, resolved)
	}

	if _, err := resolveExistingCheckout(projectPath, "missing", "UI", "package.json"); err == nil {
		t.Fatal("expected a missing checkout error")
	}
}

func TestEmbeddedComposeContract(t *testing.T) {
	files := []string{
		"docker-compose.dev.yml",
		"docker-compose.prod.yml",
		"docker-compose.hybrid-ui.yml",
		"docker-compose.hybrid-core.yml",
	}
	for _, name := range files {
		t.Run(name, func(t *testing.T) {
			embedded, err := embeddedComposeFiles.ReadFile("docker/" + name)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(embedded), "version: '3.8'") {
				t.Fatal("obsolete Compose version declaration is present")
			}
			if strings.Contains(string(embedded), ":latest") {
				t.Fatal("runtime image is not pinned")
			}

			shipped, err := os.ReadFile(filepath.Join("..", "docker", name))
			if err != nil {
				t.Fatal(err)
			}
			if string(embedded) != string(shipped) {
				t.Fatal("embedded and repository Compose files differ")
			}
		})
	}

	prod, err := embeddedComposeFiles.ReadFile("docker/docker-compose.prod.yml")
	if err != nil {
		t.Fatal(err)
	}
	content := string(prod)
	for _, expected := range []string{
		"KUBEORCH_MONGO_URI: mongodb://mongodb:27017/kubeorchestra",
		`- "3001:3000"`,
		"NEXT_PUBLIC_API_URL: http://localhost:3000/v1/api",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("production Compose is missing %q", expected)
		}
	}
}
