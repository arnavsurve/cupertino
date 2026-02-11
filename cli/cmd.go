package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var yesFlag = flag.Bool("y", false, "Assume yes to all prompts")

func init() {
	flag.Parse()
}

func confirmAction(message string) bool {
	if *yesFlag {
		return true
	}

	fmt.Printf("%s (y/N): ", message)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		response := strings.ToLower(strings.TrimSpace(scanner.Text()))
		return response == "y" || response == "yes" || response == "Y" || response == "Yes" || response == "YES"
	}
	return false
}

func uninstall(args []string) {
	packageName := args[0]

	db, err := NewSQLitePackageDB(getDatabasePath())
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		return
	}
	defer db.Close()

	if !db.HasAnyVersion(packageName) {
		fmt.Printf("Package '%s' is not installed\n", packageName)
		return
	}

	dependents, err := db.GetDependents(packageName)
	if err != nil {
		fmt.Printf("Error checking dependencies: %v\n", err)
		return
	}

	if len(dependents) > 0 {
		fmt.Printf("Cannot uninstall '%s' - the following packages are dependents:\n", packageName)
		for _, dep := range dependents {
			fmt.Printf("  - %s %s\n", dep.Name, dep.Version)
		}

		if !confirmAction("Uninstall anyway? (This may break dependent packages)") {
			fmt.Println("Uninstall cancelled.")
			return
		}
	} else if !confirmAction(fmt.Sprintf("Remove %s?", packageName)) {
		fmt.Println("Uninstall cancelled.")
		return
	}

	pkg, err := db.Get(packageName)
	if err != nil {
		fmt.Printf("Error getting package info: %v\n", err)
		return
	}

	fmt.Printf("Uninstalling %s v%s...\n", pkg.Name, pkg.Version)

	dirsToCleanup := make(map[string]bool)
	filesRemoved := 0

	for _, filePath := range pkg.InstalledFiles {
		if err := os.Remove(filePath); err != nil {
			if !os.IsNotExist(err) {
				fmt.Printf("Warning: could not remove %s: %v\n", filePath, err)
			}
		} else {
			fmt.Printf("Removed %s\n", filePath)
			filesRemoved++

			dirsToCleanup[filepath.Dir(filePath)] = true
		}
	}

	for dirPath := range dirsToCleanup {
		cleanupEmptyDirs(dirPath)
	}

	removeSymlinks(pkg)

	if err := db.Remove(packageName); err != nil {
		fmt.Printf("Error deleting from database: %v\n", err)
		return
	}

	fmt.Printf("✅ Successfully uninstalled %s (%d files)\n", packageName, filesRemoved)
}

func list() {
	db, err := NewSQLitePackageDB(getDatabasePath())
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		return
	}
	defer db.Close()

	packages, err := db.List()
	if err != nil {
		fmt.Printf("Error listing packages: %v\n", err)
		return
	}

	if len(packages) == 0 {
		fmt.Println("No packages installed")
		return
	}

	fmt.Printf("Installed packages (%d total):\n", len(packages))
	for _, pkg := range packages {
		installDate := pkg.InstallDate.Format("2006-01-02")

		fmt.Printf("  %-20s %-10s (installed %s)\n",
			pkg.Name, pkg.Version, installDate)

		if pkg.Description != "" {
			fmt.Printf("    %s\n", pkg.Description)
		}
	}
}

func search(query string) {
	registryURL := getRegistryURL()
	url := fmt.Sprintf("%s/api/search?q=%s&limit=20", registryURL, query)

	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("Error: registry returned HTTP %d\n", resp.StatusCode)
		return
	}

	var results []RegistryPackageInfo
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		fmt.Printf("Error parsing results: %v\n", err)
		return
	}

	if len(results) == 0 {
		fmt.Printf("No packages found for '%s'\n", query)
		return
	}

	for _, pkg := range results {
		fmt.Printf("  %-20s %-10s %s\n", pkg.Name, pkg.Latest, pkg.Description)
	}
}

func info(packageName string) {
	registryURL := getRegistryURL()
	pkgInfo, err := getPackageInfo(registryURL, packageName)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("%s\n", pkgInfo.Name)
	if pkgInfo.Description != "" {
		fmt.Printf("  %s\n", pkgInfo.Description)
	}
	fmt.Println()

	fmt.Printf("  latest:    %s\n", pkgInfo.Latest)
	fmt.Printf("  downloads: %d\n", pkgInfo.Downloads)
	if pkgInfo.License != "" {
		fmt.Printf("  license:   %s\n", pkgInfo.License)
	}
	if pkgInfo.Homepage != "" {
		fmt.Printf("  homepage:  %s\n", pkgInfo.Homepage)
	}

	fmt.Printf("\n  versions:  %s\n", strings.Join(pkgInfo.Versions, ", "))

	// Check if installed locally
	db, err := NewSQLitePackageDB(getDatabasePath())
	if err == nil {
		defer db.Close()
		if db.HasAnyVersion(packageName) {
			installedVersion, _ := db.GetInstalledVersion(packageName)
			fmt.Printf("\n  installed: %s\n", installedVersion)
		}
	}
}

func upgrade(packageName string) {
	registryURL := getRegistryURL()

	db, err := NewSQLitePackageDB(getDatabasePath())
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		return
	}
	defer db.Close()

	if !db.HasAnyVersion(packageName) {
		fmt.Printf("Package '%s' is not installed\n", packageName)
		return
	}

	installedVersion, err := db.GetInstalledVersion(packageName)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	pkgInfo, err := getPackageInfo(registryURL, packageName)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if pkgInfo.Latest == installedVersion {
		fmt.Printf("%s is already up to date (v%s)\n", packageName, installedVersion)
		return
	}

	fmt.Printf("%s: %s -> %s\n", packageName, installedVersion, pkgInfo.Latest)

	if !confirmAction("Upgrade?") {
		fmt.Println("Upgrade cancelled.")
		return
	}

	if err := installFromRegistry(packageName + "@" + pkgInfo.Latest); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

func upgradeAll() {
	db, err := NewSQLitePackageDB(getDatabasePath())
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		return
	}
	defer db.Close()

	packages, err := db.List()
	if err != nil {
		fmt.Printf("Error listing packages: %v\n", err)
		return
	}

	if len(packages) == 0 {
		fmt.Println("No packages installed")
		return
	}

	registryURL := getRegistryURL()
	var upgradeable []struct{ name, from, to string }

	for _, pkg := range packages {
		pkgInfo, err := getPackageInfo(registryURL, pkg.Name)
		if err != nil {
			continue
		}
		if pkgInfo.Latest != pkg.Version {
			upgradeable = append(upgradeable, struct{ name, from, to string }{
				pkg.Name, pkg.Version, pkgInfo.Latest,
			})
		}
	}

	if len(upgradeable) == 0 {
		fmt.Println("All packages are up to date")
		return
	}

	fmt.Printf("%d package(s) can be upgraded:\n", len(upgradeable))
	for _, u := range upgradeable {
		fmt.Printf("  %-20s %s -> %s\n", u.name, u.from, u.to)
	}

	if !confirmAction("Upgrade all?") {
		fmt.Println("Upgrade cancelled.")
		return
	}

	for _, u := range upgradeable {
		fmt.Printf("\nUpgrading %s...\n", u.name)
		if err := installFromRegistry(u.name + "@" + u.to); err != nil {
			fmt.Printf("Error upgrading %s: %v\n", u.name, err)
		}
	}
}

func brewInstall(args []string) {
	if len(args) == 0 {
		fmt.Println("Error: cupertino brew install requires a package name")
		return
	}

	packageName := args[0]
	fmt.Printf("Fetching Homebrew formula for %s...\n", packageName)

	formula, err := fetchHomebrewFormula(packageName)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Found %s v%s\n", formula.Name, formula.Versions.Stable)
	fmt.Printf("Dependencies: %v\n", formula.Dependencies)

	bottleURL, checksum, err := getBottleURL(formula)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	bottlePath, err := downloadBottle(bottleURL, checksum)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer os.Remove(bottlePath)

	pkg := convertToPackage(formula)

	if err := installBottle(bottlePath, pkg); err != nil {
		fmt.Printf("Error installing bottle: %v\n", err)
		return
	}

	fmt.Printf("✅ Successfully installed %s v%s from Homebrew\n", pkg.Name, pkg.Version)
}

func showUsage() {
	fmt.Println(`
 ________      ___  ___      ________    _______       ________      _________    ___      ________       ________     
|\   ____\    |\  \|\  \    |\   __  \  |\  ___ \     |\   __  \    |\___   ___| |\  \    |\   ___  \    |\   __  \    
\ \  \___|    \ \  \\\  \   \ \  \|\  \ \ \   __/|    \ \  \|\  \   \|___ \  \_| \ \  \   \ \  \\ \  \   \ \  \|\  \   
 \ \  \        \ \  \\\  \   \ \   ____\ \ \  \_|/__   \ \   _  _\       \ \  \   \ \  \   \ \  \\ \  \   \ \  \\\  \  
  \ \  \____    \ \  \\\  \   \ \  \___|  \ \  \_|\ \   \ \  \\  \|       \ \  \   \ \  \   \ \  \\ \  \   \ \  \\\  \ 
   \ \_______\   \ \_______\   \ \__\      \ \_______\   \ \__\\ _\        \ \__\   \ \__\   \ \__\\ \__\   \ \_______\
    \|_______|    \|_______|    \|__|       \|_______|    \|__|\|__|        \|__|    \|__|    \|__| \|__|    \|_______|`)
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  cupertino install <package>    Install a package")
	fmt.Println("  cupertino uninstall <package>  Remove a package")
	fmt.Println("  cupertino search <query>       Search for packages")
	fmt.Println("  cupertino info <package>       Show package details")
	fmt.Println("  cupertino upgrade [package]    Upgrade packages")
	fmt.Println("  cupertino list                 List installed packages")
	fmt.Println("  cupertino init                 Create a package.json")
	fmt.Println("  cupertino publish              Publish a package")
	fmt.Println("  cupertino help                 Show this help")
}

func showVersion() {
	fmt.Println("cupertino v1.0.0")
}
