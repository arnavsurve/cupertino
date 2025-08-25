package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func extractTarGz(tarballPath, destDir string) error {
	file, err := os.Open(tarballPath)
	if err != nil {
		return fmt.Errorf("opening tarball: %v", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("creating gzip reader: %v", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return fmt.Errorf("reading tar header: %v", err)
		}

		// Skip directories, we create them as needed
		if header.Typeflag == tar.TypeDir {
			continue
		}

		destPath := filepath.Join(destDir, header.Name)

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return fmt.Errorf("creating directory: %v", err)
		}

		destFile, err := os.Create(destPath)
		if err != nil {
			return fmt.Errorf("creating file %s: %v", destPath, err)
		}

		_, err = io.Copy(destFile, tarReader)
		destFile.Close()

		if err != nil {
			return fmt.Errorf("copying file contents: %v", err)
		}

		if err := os.Chmod(destPath, os.FileMode(header.Mode)); err != nil {
			return fmt.Errorf("setting permissions: %v", err)
		}
	}

	return nil
}

func copyFile(src, dest string) error {
	destDir := filepath.Dir(dest)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("creating destination directory: %v", err)
	}

	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening source file: %v", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("creating destination file: %v", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("copying file contents: %v", err)
	}

	sourceStat, err := sourceFile.Stat()
	if err != nil {
		return fmt.Errorf("getting source file info: %v", err)
	}

	err = os.Chmod(dest, sourceStat.Mode())
	if err != nil {
		return fmt.Errorf("setting destination file permissions: %v", err)
	}

	return nil
}