package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func getCupertinoDir() string {
	return "/opt/cupertino"
}

func getDatabasePath() string {
	return filepath.Join(getCupertinoDir(), "packages.db")
}

func cleanupEmptyDirs(startPath string) {
	dir := startPath

	for {
		cupertinoDir := getCupertinoDir()
		if dir == cupertinoDir || dir == filepath.Dir(cupertinoDir) {
			break
		}

		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			break
		}

		if err := os.Remove(dir); err != nil {
			break
		}

		fmt.Printf("🗑️ Removed empty directory %s\n", dir)

		dir = filepath.Dir(dir)
	}
}

func getBinDir() string {
	return filepath.Join(getCupertinoDir(), "bin")
}

func createSymlinks(pkg *InstalledPackage) error {
	binDir := getBinDir()

	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("creating bin directory: %v", err)
	}

	var createdSymlinks []string

	for _, filePath := range pkg.InstalledFiles {
		if strings.Contains(filePath, "/bin/") {
			binaryName := filepath.Base(filePath)
			symlinkPath := filepath.Join(binDir, binaryName)

			// Remove existing symlink if exists
			os.Remove(symlinkPath)

			if err := os.Symlink(filePath, symlinkPath); err != nil {
				for _, link := range createdSymlinks {
					os.Remove(link)
				}
				return fmt.Errorf("creating symlink %s: %v", symlinkPath, err)
			}

			createdSymlinks = append(createdSymlinks, symlinkPath)
			fmt.Printf("🔗 Linked %s -> %s\n", binaryName, symlinkPath)
		}
	}

	return nil
}

func removeSymlinks(pkg *InstalledPackage) {
	binDir := getBinDir()

	for _, filePath := range pkg.InstalledFiles {
		if strings.Contains(filePath, "/bin/") {
			binaryName := filepath.Base(filePath)
			symlinkPath := filepath.Join(binDir, binaryName)

			if target, err := os.Readlink(symlinkPath); err == nil && target == filePath {
				if err := os.Remove(symlinkPath); err == nil {
					fmt.Printf("⛓️‍💥 Removed symlink %s\n", binaryName)
				}
			}
		}
	}
}

func showPathInstructions() {
	binDir := getBinDir()

	fmt.Println("\n💡 To use installed programs, add this to your shell profile:")
	fmt.Printf("   export PATH=\"%s:$PATH\"\n", binDir)
	fmt.Println("\nThen restart your shell or run:")
	fmt.Printf("   `source ~/.zshrc` or `source ~/.bashrc`\n\n")
}

func detectPlatform() string {
	if runtime.GOOS != "darwin" {
		if runtime.GOARCH == "arm64" {
			return "arm64_linux"
		}
		return "x86_64_linux"
	}

	if runtime.GOARCH == "arm64" {
		return "arm64_sonoma"
	}
	return "sonoma"
}
