package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const defaultRegistry = "http://localhost:8080"

type RegistryPackage struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Description  string            `json:"description"`
	Homepage     string            `json:"homepage"`
	License      string            `json:"license"`
	Dependencies map[string]string `json:"dependencies"`
	Files        map[string]string `json:"files"`
	Checksum     string            `json:"checksum"`
	Size         int64             `json:"size"`
	DownloadURL  string            `json:"download_url"`
}

type RegistryPackageInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Homepage    string   `json:"homepage"`
	License     string   `json:"license"`
	Versions    []string `json:"versions"`
	Latest      string   `json:"latest"`
	Downloads   int      `json:"downloads"`
}

func installFromRegistry(packageSpec string) error {
	name, version := parsePackageSpec(packageSpec)
	registryURL := getRegistryURL()

	fmt.Printf("Fetching package info for %s...\n", name)

	var rootPkg *Package
	var err error

	if version == "" {
		regPkg, err := getLatestPackage(registryURL, name)
		if err != nil {
			return fmt.Errorf("failed to get package info: %v", err)
		}
		rootPkg = &Package{
			Name:         regPkg.Name,
			Version:      regPkg.Version,
			Description:  regPkg.Description,
			Homepage:     regPkg.Homepage,
			License:      regPkg.License,
			Dependencies: regPkg.Dependencies,
			Files:        regPkg.Files,
		}
	} else {
		regPkg, err := getSpecificPackage(registryURL, name, version)
		if err != nil {
			return fmt.Errorf("failed to get package info: %v", err)
		}
		rootPkg = &Package{
			Name:         regPkg.Name,
			Version:      regPkg.Version,
			Description:  regPkg.Description,
			Homepage:     regPkg.Homepage,
			License:      regPkg.License,
			Dependencies: regPkg.Dependencies,
			Files:        regPkg.Files,
		}
	}

	fmt.Printf("Building dependency tree for %s v%s...\n", rootPkg.Name, rootPkg.Version)

	result, err := ResolveDependencies(rootPkg)
	if err != nil {
		return fmt.Errorf("dependency resolution failed: %v", err)
	}

	fmt.Printf("Found %d packages to install:\n", len(result.Packages))
	for _, pkg := range result.Packages {
		fmt.Printf("  %s v%s\n", pkg.Name, pkg.Version)
	}

	for _, pkg := range result.Packages {
		shouldInstall, reason, err := evaluateInstallationNeed(pkg.Name, pkg.Version)
		if err != nil {
			return fmt.Errorf("failed to evalurate installation need for %s: %v", pkg.Name, err)
		}

		if !shouldInstall {
			fmt.Printf("Skipping %s v%s (%s)\n", pkg.Name, pkg.Version, reason)
			continue
		}

		fmt.Printf("Installing %s v%s...\n", pkg.Name, pkg.Version)

		// If replacing existing version, remove it first
		if strings.Contains(reason, "upgrade") || strings.Contains(reason, "downgrade") || strings.Contains(reason, "replace") {
			fmt.Printf("Removing previous version of %s...\n", pkg.Name)
			if err := removePackageByName(pkg.Name); err != nil {
				fmt.Printf("Warning: failed to remove old version: %v\n", err)
			}
		}

		regPkg, err := getSpecificPackage(registryURL, pkg.Name, pkg.Version)
		if err != nil {
			return fmt.Errorf("failed to get download info for %s: %v", pkg.Name, err)
		}

		tempFile, err := downloadAndVerify(regPkg.DownloadURL, regPkg.Checksum)
		if err != nil {
			return fmt.Errorf("failed to download %s: %v", pkg.Name, err)
		}
		defer os.Remove(tempFile)

		if err := installFromTarball(tempFile); err != nil {
			return fmt.Errorf("failed to install %s: %v", pkg.Name, err)
		}
	}

	return nil
}

func parsePackageSpec(spec string) (name, version string) {
	if strings.Contains(spec, "@") {
		parts := strings.SplitN(spec, "@", 2)
		return parts[0], parts[1]
	}

	return spec, ""
}

func getRegistryURL() string {
	if url := os.Getenv("CUPERTINO_REGISTRY"); url != "" {
		return url
	}

	return defaultRegistry
}

func getLatestPackage(registryURL, name string) (*RegistryPackage, error) {
	url := fmt.Sprintf("%s/api/packages/%s", registryURL, name)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("package '%s' not found", name)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("registry error: HTTP %d", resp.StatusCode)
	}

	var info RegistryPackageInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to parse package info: %v", err)
	}

	return getSpecificPackage(registryURL, name, info.Latest)
}

func getSpecificPackage(registryURL, name, version string) (*RegistryPackage, error) {
	url := fmt.Sprintf("%s/api/packages/%s/%s", registryURL, name, version)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("package '%s' version '%s' not found", name, version)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("registry error: HTTP %d", resp.StatusCode)
	}

	var pkg RegistryPackage
	if err := json.NewDecoder(resp.Body).Decode(&pkg); err != nil {
		return nil, fmt.Errorf("failed to parse package: %v", err)
	}

	return &pkg, nil
}

func downloadAndVerify(url, expectedChecksum string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	tempFile, err := os.CreateTemp("", "cupertino-download-*.tar.gz")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %v", err)
	}

	tempPath := tempFile.Name()

	hasher := sha256.New()
	writer := io.MultiWriter(tempFile, hasher)

	_, err = io.Copy(writer, resp.Body)
	tempFile.Close()

	if err != nil {
		os.Remove(tempPath)
		return "", fmt.Errorf("download failed: %v", err)
	}

	actualChecksum := fmt.Sprintf("%x", hasher.Sum(nil))
	if actualChecksum != expectedChecksum {
		os.Remove(tempPath)
		return "", fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	return tempPath, nil
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"B", "KB", "MB", "GB"}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp+1])
}

func getPackageInfo(registryURL, packageName string) (*RegistryPackageInfo, error) {
	url := fmt.Sprintf("%s/api/packages/%s", registryURL, packageName)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("package '%s' not found", packageName)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("registry error: HTTP %d", resp.StatusCode)
	}

	var info RegistryPackageInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to parse package info: %v", err)
	}

	return &info, nil
}

func evaluateInstallationNeed(name, targetVersion string) (bool, string, error) {
	db, err := NewSQLitePackageDB(getDatabasePath())
	if err != nil {
		return false, "", err
	}
	defer db.Close()

	if db.IsInstalledVersion(name, targetVersion) {
		return false, "already installed", nil
	}

	if !db.HasAnyVersion(name) {
		return true, "new installation", nil
	}

	installedVersion, err := db.GetInstalledVersion(name)
	if err != nil {
		return true, "installation check failed", nil
	}

	currentVer, err1 := ParseVersion(installedVersion)
	newVer, err2 := ParseVersion(targetVersion)

	if err1 != nil || err2 != nil {
		return true, fmt.Sprintf("replace %s", installedVersion), nil
	}

	comparison := newVer.Compare(currentVer)
	if comparison > 0 {
		return true, fmt.Sprintf("upgrade from %s", installedVersion), nil
	} else if comparison < 0 {
		return true, fmt.Sprintf("downgrade from %s", installedVersion), nil
	}

	return false, "same version already installed", nil
}

func satisfiesDependencyConstraint(packageName, constraint string) (bool, error) {
	db, err := NewSQLitePackageDB(getDatabasePath())
	if err != nil {
		return false, err
	}
	defer db.Close()

	if !db.HasAnyVersion(packageName) {
		return false, nil
	}

	installedVersion, err := db.GetInstalledVersion(packageName)
	if err != nil {
		return false, err
	}

	parsedConstraint, err := ParseConstraint(constraint)
	if err != nil {
		return false, err
	}

	installedVer, err := ParseVersion(installedVersion)
	if err != nil {
		return false, err
	}

	return parsedConstraint.Satisfies(installedVer), nil
}

func removePackageByName(name string) error {
	db, err := NewSQLitePackageDB(getDatabasePath())
	if err != nil {
		return err
	}
	defer db.Close()

	if !db.HasAnyVersion(name) {
		return fmt.Errorf("package %s is not installed", name)
	}

	pkg, err := db.Get(name)
	if err != nil {
		return err
	}

	removeSymlinks(pkg)

	filesRemoved := 0
	for _, filePath := range pkg.InstalledFiles {
		if err := os.Remove(filePath); err == nil {
			filesRemoved++
		}
	}

	cleanupEmptyDirs(pkg.InstallPath)

	err = db.Remove(name)
	if err != nil {
		return err
	}

	fmt.Printf("Removed %s (%d files)\n", name, filesRemoved)
	return nil
}
