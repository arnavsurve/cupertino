package main

import (
	"fmt"
	"strconv"
	"strings"
)

type Version struct {
	Major int
	Minor int
	Patch int
}

type VersionConstraint struct {
	Operator string // >=, ^, ~, =, *
	Version  Version
}

func ParseVersion(versionStr string) (Version, error) {
	parts := strings.Split(versionStr, ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("invalid version format: %s", versionStr)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return Version{}, fmt.Errorf("invalid major version: %s", parts[0])
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return Version{}, fmt.Errorf("invalid minor version: %s", parts[1])
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return Version{}, fmt.Errorf("invalid patch version: %s", parts[2])
	}

	return Version{
		Major: major,
		Minor: minor,
		Patch: patch,
	}, nil
}

func ParseConstraint(constraintStr string) (VersionConstraint, error) {
	if constraintStr == "*" {
		return VersionConstraint{Operator: "*"}, nil
	}

	if after, ok := strings.CutPrefix(constraintStr, ">="); ok {
		version, err := ParseVersion(after)
		return VersionConstraint{Operator: ">=", Version: version}, err
	}

	if after, ok := strings.CutPrefix(constraintStr, "^"); ok {
		version, err := ParseVersion(after)
		return VersionConstraint{Operator: "^", Version: version}, err
	}

	if after, ok := strings.CutPrefix(constraintStr, "~"); ok {
		version, err := ParseVersion(after)
		return VersionConstraint{Operator: "~", Version: version}, err
	}

	// Default to exact match
	version, err := ParseVersion(constraintStr)
	return VersionConstraint{Operator: "=", Version: version}, err
}

func (v Version) Compare(other Version) int {
	if v.Major != other.Major {
		return v.Major - other.Major
	}

	if v.Minor != other.Minor {
		return v.Minor - other.Minor
	}

	return v.Patch - other.Patch
}

func (constraint VersionConstraint) Satisfies(version Version) bool {
	switch constraint.Operator {
	case "*":
		return true
	case "=":
		return constraint.Version.Compare(version) == 0
	case ">=":
		return version.Compare(constraint.Version) >= 0
	case "^":
		// Compatible within same major version
		return version.Major == constraint.Version.Major &&
			version.Compare(constraint.Version) >= 0
	case "~":
		// Compatible within same major.minor
		return version.Major == constraint.Version.Major &&
			version.Minor == constraint.Version.Minor &&
			version.Compare(constraint.Version) >= 0
	}

	return false
}
