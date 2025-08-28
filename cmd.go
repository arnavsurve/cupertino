package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func install(args []string) {
	packagePath := args[0]

	if _, err := os.Stat(packagePath); os.IsNotExist(err) {
		fmt.Printf("Error: package file '%s' not found\n", packagePath)
		return
	}

	fmt.Printf("Installing package from %s...\n", packagePath)

	if err := installFromTarball(packagePath); err != nil {
		fmt.Printf("Error installing package: %v\n", err)
		return
	}

	fmt.Println("✅ Package installed successfully.")
}

func uninstall(args []string) {
	packageName := args[0]

	db, err := NewSQLitePackageDB(getDatabasePath())
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		return
	}
	defer db.Close()

	if !db.IsInstalled(packageName) {
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

func testDatabase() {
	db, err := NewSQLitePackageDB("/tmp/test_cupertino.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	testPkg := &InstalledPackage{
		Package: Package{
			Name:        "test-package",
			Version:     "1.0.0",
			Description: "A test package",
			Dependencies: map[string]string{
				"git": ">=2.0",
			},
		},
		InstallPath:    "/Users/test/.cupertino/packages/test-package/1.0.0",
		InstalledFiles: []string{"bin/test", "share/man/man1/test.1"},
		InstallDate:    time.Now(),
	}

	if err := db.Install(testPkg); err != nil {
		panic(err)
	}

	if !db.IsInstalled("test-package") {
		panic("Package should be installed")
	}

	retrieved, err := db.Get("test-package")
	if err != nil {
		panic(err)
	}

	fmt.Printf("Retrieved package: %s v%s\n", retrieved.Name, retrieved.Version)
	fmt.Printf("Files: %v\n", retrieved.InstalledFiles)

	packages, err := db.List()
	if err != nil {
		panic(err)
	}

	fmt.Printf("Total packages: %d\n", len(packages))
}

func testVersions() {
	// Test version parsing
	v1, _ := ParseVersion("1.2.3")
	v2, _ := ParseVersion("1.3.0")
	v3, _ := ParseVersion("2.0.0")

	fmt.Printf("1.2.3 vs 1.3.0: %d\n", v1.Compare(v2)) // Should be -1
	fmt.Printf("1.3.0 vs 1.2.3: %d\n", v2.Compare(v1)) // Should be 1
	fmt.Printf("1.2.3 vs 1.2.3: %d\n", v1.Compare(v1)) // Should be 0

	// Test constraints
	constraint1, _ := ParseConstraint(">=1.2.0")
	constraint2, _ := ParseConstraint("^1.2.0")
	constraint3, _ := ParseConstraint("~1.2.0")

	fmt.Printf(">=1.2.0 satisfies 1.2.3: %v\n", constraint1.Satisfies(v1)) // true
	fmt.Printf(">=1.2.0 satisfies 2.0.0: %v\n", constraint1.Satisfies(v3)) // true
	fmt.Printf("^1.2.0 satisfies 1.3.0: %v\n", constraint2.Satisfies(v2))  // true
	fmt.Printf("^1.2.0 satisfies 2.0.0: %v\n", constraint2.Satisfies(v3))  // false
	fmt.Printf("~1.2.0 satisfies 1.2.3: %v\n", constraint3.Satisfies(v1))  // true
	fmt.Printf("~1.2.0 satisfies 1.3.0: %v\n", constraint3.Satisfies(v2))  // false
}

func testDependencyResolution() {
	// Create a test package that depends on git
	testPkg := &Package{
		Name:    "test-app",
		Version: "1.0.0",
		Dependencies: map[string]string{
			"git":  ">=2.40.0",
			"node": "^18.0.0",
		},
	}

	fmt.Println("Resolving dependencies for test-app...")
	result, err := ResolveDependencies(testPkg)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("Install order:")
	for i, name := range result.Order {
		fmt.Printf("%d. %s\n", i+1, name)
	}

	fmt.Println("\nResolved packages:")
	for _, pkg := range result.Packages {
		fmt.Printf("- %s@%s\n", pkg.Name, pkg.Version)
	}
}
