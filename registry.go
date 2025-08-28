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

	var pkg *RegistryPackage
	var err error

	if version == "" {
		pkg, err = getLatestPackage(registryURL, name)
	} else {
		pkg, err = getSpecificPackage(registryURL, name, version)
	}

	if err != nil {
		return fmt.Errorf("failed to get package info: %v", err)
	}

	fmt.Printf("Found %s v%s (%s)\n", pkg.Name, pkg.Version, formatBytes(pkg.Size))

	fmt.Printf("Downloading %s...\n", pkg.DownloadURL)
	tempFile, err := downloadAndVerify(pkg.DownloadURL, pkg.Checksum)
	if err != nil {
		return fmt.Errorf("failed to download package: %v", err)
	}
	defer os.Remove(tempFile)

	fmt.Printf("Installing %s v%s...\n", pkg.Name, pkg.Version)

	return installFromTarball(tempFile)
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
