package main

import (
	"fmt"
	"os"
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
			fmt.Println("Usage: cpt install <package>")
			return
		}
		install(os.Args[2:])
	case "uninstall":
		if len(os.Args) < 3 {
			fmt.Println("Error: uninstall requires a package name")
			fmt.Println("Usage: cpt uninstall <package>")
			return
		}
		uninstall(os.Args[2:])
	case "list":
		list()
	case "help", "--help", "-h":
		showUsage()
	case "debug":
		testDatabase()
		return
	default:
		fmt.Printf("Unknown command: %s\n", command)
		showUsage()
	}
}

