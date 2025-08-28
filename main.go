package main

import (
	"flag"
	"fmt"
	"strings"
)

func main() {
	flag.Parse()
	args := flag.Args()

	if len(args) == 0 {
		showUsage()
		return
	}

	command := args[0]
	switch command {
	case "install":
		if len(args) < 2 {
			fmt.Println("Error: install requires a package name")
			fmt.Println("Usage: cupertino install <package>")
			return
		}

		packageArg := args[1]
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
		if len(args) < 2 {
			fmt.Println("Error: uninstall requires a package name")
			fmt.Println("Usage: cupertino uninstall <package>")
			return
		}
		uninstall(args[1:])
	case "list":
		list()
	case "help", "--help", "-h":
		showUsage()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		showUsage()
	}
}
