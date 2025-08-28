package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		showUsage()
		return
	}

	command := os.Args[1]
	switch command {
	case "install":
		if len(os.Args) < 3 {
			fmt.Println("Error: install requires a package name")
			fmt.Println("Usage: cupertino install <package>")
			return
		}

		packageArg := os.Args[2]

		if strings.HasPrefix(packageArg, ".tar.gz") || strings.Contains(packageArg, "/") {
			// Local file
			err := installFromTarball(packageArg)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		} else {
			// Registry package
			err := installFromRegistry(packageArg)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		}
	// case "brew":
	// 	if len(os.Args) < 3 {
	// 		fmt.Println("Error: brew requires a subcommand")
	// 		fmt.Println("Usage: cupertino brew <subcommand>")
	// 		return
	// 	}
	// 	subcommand := os.Args[2]
	// 	switch subcommand {
	// 	case "install":
	// 		brewInstall(os.Args[3:])
	// 	default:
	// 		fmt.Printf("Unknown command: %s\n", command)
	// 	}
	case "uninstall":
		if len(os.Args) < 3 {
			fmt.Println("Error: uninstall requires a package name")
			fmt.Println("Usage: cupertino uninstall <package>")
			return
		}
		uninstall(os.Args[2:])
	case "list":
		list()
	case "help", "--help", "-h":
		showUsage()
	case "test-versions":
		testVersions()
	case "test-deps":
		testDependencyResolution()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		showUsage()
	}
}
