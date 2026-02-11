package main

import "time"

type Package struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Homepage    string `json:"homepage,omitempty"`
	License     string `json:"license,omitempty"`

	Dependencies map[string]string `json:"dependencies,omitempty"` // "git": ">=2.0"

	// Files to install - source path -> destination path
	Files map[string]string `json:"files"`

	// Scripts to run during installation
	PreInstall  []string `json:"pre_install,omitempty"`
	PostInstall []string `json:"post_install,omitempty"`
	PreRemove   []string `json:"pre_remove,omitempty"`
	PostRemove  []string `json:"post_remove,omitempty"`
}

type InstalledPackage struct {
	Package
	InstallPath    string    `json:"install_path"`
	InstalledFiles []string  `json:"installed_files"`
	InstallDate    time.Time `json:"install_date"`
}
