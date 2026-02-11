package main

import (
	"fmt"
)

type ResolutionResult struct {
	Packages []*Package
	Order    []string // Package names in install order
}

func ResolveDependencies(rootPackage *Package) (*ResolutionResult, error) {
	resolved := make([]*Package, 0)
	visited := make(map[string]bool)
	visiting := make(map[string]bool)
	order := make([]string, 0)

	err := resolveDepsRecursive(rootPackage, &resolved, &order, visited, visiting)
	if err != nil {
		return nil, err
	}

	return &ResolutionResult{
		Packages: resolved,
		Order:    order,
	}, nil
}

func resolveDepsRecursive(
	pkg *Package,
	resolved *[]*Package,
	order *[]string,
	visited,
	visiting map[string]bool,
) error {
	pkgKey := pkg.Name + "@" + pkg.Version

	// Check circular dependency
	if visiting[pkgKey] {
		return fmt.Errorf("circular dependency detected: %s", pkg.Name)
	}

	// Already processed
	if visited[pkgKey] {
		return nil
	}

	visiting[pkgKey] = true

	for depName, constraintStr := range pkg.Dependencies {
		depPkg, err := fetchPackage(depName, constraintStr)
		if err != nil {
			return fmt.Errorf("resolving dependency %s: %v", depName, err)
		}

		err = resolveDepsRecursive(depPkg, resolved, order, visited, visiting)
		if err != nil {
			return err
		}
	}

	visiting[pkgKey] = false
	visited[pkgKey] = true
	*resolved = append(*resolved, pkg)
	*order = append(*order, pkg.Name)

	return nil
}

func fetchPackage(name, constraintStr string) (*Package, error) {
	satisfied, err := satisfiesDependencyConstraint(name, constraintStr)
	if err == nil && satisfied {
		db, err := NewSQLitePackageDB(getDatabasePath())
		if err == nil {
			defer db.Close()
			if installedPkg, err := db.Get(name); err == nil {
				return &installedPkg.Package, nil
			}
		}
	}

	registryURL := getRegistryURL()
	constraint, err := ParseConstraint(constraintStr)
	if err != nil {
		return nil, err
	}

	packageInfo, err := getPackageInfo(registryURL, name)
	if err != nil {
		return nil, err
	}

	var bestVersion string
	var bestVersionParsed Version

	for _, versionStr := range packageInfo.Versions {
		version, err := ParseVersion(versionStr)
		if err != nil {
			continue // Skip invalid versions
		}

		if constraint.Satisfies(version) {
			if bestVersion == "" || version.Compare(bestVersionParsed) > 0 {
				bestVersion = versionStr
				bestVersionParsed = version
			}
		}
	}

	if bestVersion == "" {
		return nil, fmt.Errorf("no version of %s satisfies constraint %s", name, constraintStr)
	}

	regPkg, err := getSpecificPackage(registryURL, name, bestVersion)
	if err != nil {
		return nil, err
	}

	return &Package{
		Name:         regPkg.Name,
		Version:      regPkg.Version,
		Description:  regPkg.Description,
		Homepage:     regPkg.Homepage,
		License:      regPkg.License,
		Dependencies: regPkg.Dependencies,
		Files:        regPkg.Files,
	}, nil
}
