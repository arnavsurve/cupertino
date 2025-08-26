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
	constraint, err := ParseConstraint(constraintStr)
	if err != nil {
		return nil, err
	}

	// TODO: registry lookup
	mockPackages := getMockPackages()

	for _, pkg := range mockPackages[name] {
		version, err := ParseVersion(pkg.Version)
		if err != nil {
			continue
		}

		if constraint.Satisfies(version) {
			return pkg, nil
		}
	}
	return nil, fmt.Errorf("no version of %s satisfies constraint %s", name, constraintStr)
}

// Mock package database for testing
func getMockPackages() map[string][]*Package {
	return map[string][]*Package{
		"git": {
			{Name: "git", Version: "2.42.0", Dependencies: map[string]string{"openssl": ">=1.1.0"}},
			{Name: "git", Version: "2.41.0", Dependencies: map[string]string{"openssl": ">=1.0.0"}},
		},
		"openssl": {
			{Name: "openssl", Version: "3.0.0", Dependencies: map[string]string{}},
			{Name: "openssl", Version: "1.1.1", Dependencies: map[string]string{}},
		},
		"node": {
			{Name: "node", Version: "18.17.0", Dependencies: map[string]string{"openssl": "^3.0.0"}},
		},
	}
}
