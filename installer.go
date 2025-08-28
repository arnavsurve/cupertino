package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

func installFromTarball(tarballPath string) error {
	tempDir, err := os.MkdirTemp("", "cupertino-install-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fmt.Printf("📦 Extracting to %s...\n", tempDir)

	if err := extractTarGz(tarballPath, tempDir); err != nil {
		return fmt.Errorf("extracting tarball: %v", err)
	}

	manifestPath := filepath.Join(tempDir, "package.json")
	pkg, err := parsePackageManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("parsing package.json: %v", err)
	}

	fmt.Printf("Installing %s v%s...\n", pkg.Name, pkg.Version)

	packageDir := filepath.Join(getCupertinoDir(), "packages", pkg.Name, pkg.Version)
	if err := os.MkdirAll(packageDir, 0755); err != nil {
		return fmt.Errorf("creating package dir: %v", err)
	}

	var installedFiles []string
	for srcPath, destPath := range pkg.Files {
		src := filepath.Join(tempDir, srcPath)
		dest := filepath.Join(packageDir, destPath)

		if err := copyFile(src, dest); err != nil {
			return fmt.Errorf("copying %s: %v", srcPath, err)
		}

		installedFiles = append(installedFiles, dest)
		fmt.Printf("Copied %s -> %s\n", srcPath, destPath)
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

	fmt.Printf("✅ Successfully installed %s v%s.\n", pkg.Name, pkg.Version)
	return nil
}

func parsePackageManifest(manifestPath string) (*Package, error) {
	file, err := os.Open(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("opening package.json: %v", err)
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("reading package.json: %v", err)
	}

	var pkg Package
	if err = json.Unmarshal(bytes, &pkg); err != nil {
		return nil, fmt.Errorf("unmarshalling package.json: %v", err)
	}

	return &pkg, nil
}
