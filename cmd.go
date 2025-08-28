package main

import (
	"bufio"
	"flag"
	"fmt"
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
			fmt.Printf("🗑️ Removed %s\n", filePath)
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
	fmt.Println("  cupertino list                 List installed packages")
	fmt.Println("  cupertino help                 Show this help")
}
