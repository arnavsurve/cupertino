package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

type HomebrewFormula struct {
	Name         string           `json:"name"`
	Description  string           `json:"description"`
	Homepage     string           `json:"homepage"`
	License      string           `json:"license"`
	Versions     HomebrewVersions `json:"versions"`
	Dependencies []string         `json:"dependencies"`
	Bottle       HomebrewBottle   `json:"bottle"`
}

type HomebrewVersions struct {
	Stable string `json:"stable"`
}

type HomebrewBottle struct {
	Stable HomebrewBottleStable `json:"stable"`
}

type HomebrewBottleStable struct {
	Files map[string]HomebrewBottleFile `json:"files"`
}

type HomebrewBottleFile struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}

func fetchHomebrewFormula(name string) (*HomebrewFormula, error) {
	url := fmt.Sprintf("https://formulae.brew.sh/api/formula/%s.json", name)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching formula: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("formula not found: %s, HTTP %d", name, resp.StatusCode)
	}

	var formula HomebrewFormula
	if err := json.NewDecoder(resp.Body).Decode(&formula); err != nil {
		return nil, fmt.Errorf("failed to parse formula: %v", err)
	}

	return &formula, nil
}

func getBottleURL(formula *HomebrewFormula) (string, string, error) {
	files := formula.Bottle.Stable.Files

	preferred := detectPlatform()
	if bottleFile, exists := files[preferred]; exists {
		return bottleFile.URL, bottleFile.SHA256, nil
	}

	var fallbacks []string

	if runtime.GOARCH == "arm64" && runtime.GOOS == "darwin" {
		fallbacks = []string{"arm64_sequoia", "arm64_sonoma", "arm64_ventura"}
	} else if runtime.GOOS == "darwin" {
		fallbacks = []string{"sonoma", "ventura"}
	} else if runtime.GOARCH == "arm64" {
		fallbacks = []string{"arm64_linux"}
	} else {
		fallbacks = []string{"x86_64_linux"}
	}

	for _, platform := range fallbacks {
		if bottleFile, exists := files[platform]; exists {
			return bottleFile.URL, bottleFile.SHA256, nil
		}
	}

	return "", "", fmt.Errorf("no compatible bottle found")
}

func convertToPackage(formula *HomebrewFormula) *Package {
	deps := make(map[string]string)
	for _, dep := range formula.Dependencies {
		deps[dep] = "*" // Accept any version for now (TODO)
	}

	return &Package{
		Name:         formula.Name,
		Version:      formula.Versions.Stable,
		Description:  formula.Description,
		Homepage:     formula.Homepage,
		License:      formula.License,
		Dependencies: deps,
	}
}

func downloadBottle(url, expectedChecksum string) (string, error) {
	tempFile, err := os.CreateTemp("", "cupertino-bottle-*.tar.gz")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %v", err)
	}
	tempPath := tempFile.Name()

	fmt.Printf("Downloading %s...\n", url)

	resp, err := http.Get(url)
	if err != nil {
		os.Remove(tempPath)
		return "", fmt.Errorf("downloading bottle: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		os.Remove(tempPath)
		return "", fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	hasher := sha256.New()
	writer := io.MultiWriter(tempFile, hasher)

	_, err = io.Copy(writer, resp.Body)
	tempFile.Close()

	if err != nil {
		os.Remove(tempPath)
		return "", fmt.Errorf("writing bottle: %v", err)
	}

	actualChecksum := fmt.Sprintf("%x", hasher.Sum(nil))
	if actualChecksum != expectedChecksum {
		os.Remove(tempPath)
		return "", fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	return tempPath, nil
}

func installBottle(bottlePath string, pkg *Package) error {
	tempDir, err := os.MkdirTemp("", "cupertino-bottle-extract-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %v", err)
	}
	defer os.Remove(tempDir)

	fmt.Println("Extracting bottle...")
	if err := extractTarGz(bottlePath, tempDir); err != nil {
		return fmt.Errorf("extracting bottle: %v", err)
	}

	packageDir := findPackageInBottle(tempDir, pkg.Name)
	if packageDir == "" {
		return fmt.Errorf("package directory not found in bottle")
	}

	return installFromExtractedDir(packageDir, pkg)
}

func findPackageInBottle(bottleDir, packageName string) string {
	paths := []string{
		filepath.Join(bottleDir, "opt", "homebrew", "Cellar", packageName),
		filepath.Join(bottleDir, "usr", "local", "Cellar", packageName),
		filepath.Join(bottleDir, "home", "linuxbrew", ".linuxbrew", "Cellar", packageName),
	}

	for _, path := range paths {
		if entries, err := os.ReadDir(path); err == nil && len(entries) > 0 {
			return filepath.Join(path, entries[0].Name())
		}
	}

	return ""
}

func installFromExtractedDir(extractedDir string, pkg *Package) error {
	packageDir := filepath.Join(getCupertinoDir(), "packages", pkg.Name, pkg.Version)
	if err := os.MkdirAll(packageDir, 0755); err != nil {
		return fmt.Errorf("creating package directory: %v", err)
	}

	var installedFiles []string
	err := filepath.Walk(extractedDir, func(srcPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if srcPath == extractedDir {
			return nil
		}

		relPath, err := filepath.Rel(extractedDir, srcPath)
		if err != nil {
			return fmt.Errorf("calculating relative path: %v", err)
		}

		destPath := filepath.Join(packageDir, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		if err := copyFile(srcPath, destPath); err != nil {
			return fmt.Errorf("copying %s: %v", relPath, err)
		}

		installedFiles = append(installedFiles, destPath)
		return nil
	})
	if err != nil {
		return fmt.Errorf("copying package files: %v", err)
	}

	db, err := NewSQLitePackageDB(getDatabasePath())
	if err != nil {
		return fmt.Errorf("opening database: %v", err)
	}
	defer db.Close()

	installedPkg := &InstalledPackage{
		Package:        *pkg,
		InstallPath:    packageDir,
		InstalledFiles: installedFiles,
		InstallDate:    time.Now(),
	}

	if err := db.Install(installedPkg); err != nil {
		return fmt.Errorf("updating database: %v", err)
	}

	if err := createSymlinks(installedPkg); err != nil {
		fmt.Printf("Warning: failed to create symlinks: %v\n", err)
	}

	fmt.Printf("Copied %d files to %s\n", len(installedFiles), packageDir)
	return nil
}
