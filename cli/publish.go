package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func initPackage() {
	reader := bufio.NewReader(os.Stdin)

	// Check if package.json already exists
	if _, err := os.Stat("package.json"); err == nil {
		fmt.Println("package.json already exists in this directory")
		return
	}

	fmt.Println("Creating a new cupertino package\n")

	name := prompt(reader, "name", filepath.Base(cwd()))
	version := prompt(reader, "version", "1.0.0")
	description := prompt(reader, "description", "")
	license := prompt(reader, "license", "MIT")
	homepage := prompt(reader, "homepage", "")

	// Scan for binaries
	files := make(map[string]string)
	if entries, err := os.ReadDir("bin"); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				path := "bin/" + e.Name()
				files[path] = path
				fmt.Printf("  found %s\n", path)
			}
		}
	}

	if len(files) == 0 {
		fmt.Println("\nNo files found in bin/. You can add files manually to package.json.")
		filePath := prompt(reader, "file path (or leave empty)", "")
		if filePath != "" {
			files[filePath] = filePath
		}
	}

	pkg := Package{
		Name:        name,
		Version:     version,
		Description: description,
		License:     license,
		Homepage:    homepage,
		Files:       files,
	}

	data, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if err := os.WriteFile("package.json", data, 0644); err != nil {
		fmt.Printf("Error writing package.json: %v\n", err)
		return
	}

	fmt.Printf("\nCreated package.json for %s v%s\n", name, version)
}

func publish(args []string) {
	dryRun := false
	for _, arg := range args {
		if arg == "--dry-run" {
			dryRun = true
		}
	}

	// Read package.json
	manifestData, err := os.ReadFile("package.json")
	if err != nil {
		fmt.Println("Error: no package.json found in current directory")
		fmt.Println("Run 'cupertino init' to create one")
		return
	}

	var pkg Package
	if err := json.Unmarshal(manifestData, &pkg); err != nil {
		fmt.Printf("Error parsing package.json: %v\n", err)
		return
	}

	// Validate
	if pkg.Name == "" || pkg.Version == "" || pkg.Description == "" {
		fmt.Println("Error: package.json must have name, version, and description")
		return
	}
	if len(pkg.Files) == 0 {
		fmt.Println("Error: package.json must have at least one file")
		return
	}

	// Check that all files exist
	for srcPath := range pkg.Files {
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			fmt.Printf("Error: file '%s' not found\n", srcPath)
			return
		}
	}

	fmt.Printf("Package: %s v%s\n", pkg.Name, pkg.Version)
	fmt.Printf("Description: %s\n", pkg.Description)
	fmt.Printf("Files:\n")
	for src, dst := range pkg.Files {
		fmt.Printf("  %s -> %s\n", src, dst)
	}

	if dryRun {
		fmt.Println("\n(dry run) Package is valid and ready to publish")
		return
	}

	// Build tarball
	tarballName := fmt.Sprintf("%s-%s.tar.gz", pkg.Name, pkg.Version)
	fmt.Printf("\nBuilding %s...\n", tarballName)

	// Collect all files to include
	filesToTar := []string{"package.json"}
	for srcPath := range pkg.Files {
		filesToTar = append(filesToTar, srcPath)
	}

	cmd := exec.Command("tar", append([]string{"-czf", tarballName}, filesToTar...)...)
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("Error creating tarball: %v\n%s\n", err, output)
		return
	}
	defer os.Remove(tarballName)

	// Get API key
	apiKey := os.Getenv("CUPERTINO_API_KEY")
	if apiKey == "" {
		fmt.Println("Error: CUPERTINO_API_KEY environment variable is required")
		fmt.Println("Set it with: export CUPERTINO_API_KEY=your-key")
		return
	}

	if !confirmAction(fmt.Sprintf("Publish %s v%s?", pkg.Name, pkg.Version)) {
		fmt.Println("Publish cancelled.")
		return
	}

	// Upload
	registryURL := getRegistryURL()
	fmt.Printf("Publishing to %s...\n", registryURL)

	if err := uploadPackage(registryURL, apiKey, tarballName, &pkg); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Published %s v%s\n", pkg.Name, pkg.Version)
}

func uploadPackage(registryURL, apiKey, tarballPath string, pkg *Package) error {
	tarball, err := os.Open(tarballPath)
	if err != nil {
		return fmt.Errorf("opening tarball: %v", err)
	}
	defer tarball.Close()

	metadata := map[string]interface{}{
		"name":        pkg.Name,
		"version":     pkg.Version,
		"description": pkg.Description,
		"files":       pkg.Files,
	}
	if pkg.Homepage != "" {
		metadata["homepage"] = pkg.Homepage
	}
	if pkg.License != "" {
		metadata["license"] = pkg.License
	}
	if len(pkg.Dependencies) > 0 {
		metadata["dependencies"] = pkg.Dependencies
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshalling metadata: %v", err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("metadata", string(metadataJSON)); err != nil {
		return fmt.Errorf("writing metadata field: %v", err)
	}

	part, err := writer.CreateFormFile("file", filepath.Base(tarballPath))
	if err != nil {
		return fmt.Errorf("creating form file: %v", err)
	}
	if _, err := io.Copy(part, tarball); err != nil {
		return fmt.Errorf("copying tarball: %v", err)
	}

	writer.Close()

	req, err := http.NewRequest("POST", registryURL+"/api/packages", &body)
	if err != nil {
		return fmt.Errorf("creating request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-API-Key", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("uploading: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 409 {
		return fmt.Errorf("%s v%s already exists in the registry", pkg.Name, pkg.Version)
	}

	if resp.StatusCode != 201 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("registry returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func prompt(reader *bufio.Reader, label, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("%s (%s): ", label, defaultVal)
	} else {
		fmt.Printf("%s: ", label)
	}

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return defaultVal
	}
	return input
}

func cwd() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	return dir
}
