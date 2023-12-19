package farwrite

import (
	"archive/tar"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

type Options struct {
	LogFormat  string
	LogLevel   string
	SourcePath string
}

var trackedPaths []string

func Execute() int {
	options := parseArgs()

	logger, err := getLogger(options.LogLevel, options.LogFormat)
	if err != nil {
		slog.Error("getLogger", "error", err)
		return 1
	}

	slog.SetDefault(logger)

	err = run(options)
	if err != nil {
		slog.Error("run", "error", err)
		return 1
	}
	printTrackedPaths()
	deleteTrackedPaths()

	return 0
}

func parseArgs() Options {
	options := Options{}

	flag.StringVar(&options.LogLevel, "log-level", "info", "Log level (debug, info, warn, error), default: info")
	flag.StringVar(&options.LogFormat, "log-format", "", "Log format (text or json)")
	flag.StringVar(&options.SourcePath, "src", "", "Source directory")

	flag.Parse()

	return options
}

func createInMemoryTar(srcPath string) ([]byte, error) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	err := filepath.Walk(srcPath, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.Contains(file, string(filepath.Separator)+".git"+string(filepath.Separator)) {
			return nil
		}

		header, err := tar.FileInfoHeader(fi, "")
		if err != nil {
			return err
		}

		absolutePath, err := filepath.Abs(file)
		if err != nil {
			return err
		}

		// Exclude top-level folder from tracking
		if absolutePath != filepath.Clean(srcPath) {
			trackedPaths = append(trackedPaths, absolutePath)
		}

		header.Name, err = filepath.Rel(srcPath, file)
		if err != nil {
			return err
		}

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if fi.Mode().IsRegular() {
			fileContent, err := os.Open(file)
			if err != nil {
				return err
			}
			defer fileContent.Close()

			if _, err := io.Copy(tw, fileContent); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func extractInMemoryTar(tarData []byte, destPath string) error {
	tr := tar.NewReader(bytes.NewReader(tarData))

	for {
		header, err := tr.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		destFilePath := filepath.Join(destPath, header.Name)

		if header.FileInfo().IsDir() {
			os.MkdirAll(destFilePath, os.ModePerm)
			continue
		}

		file, err := os.Create(destFilePath)
		if err != nil {
			return err
		}
		defer file.Close()

		if _, err := io.Copy(file, tr); err != nil {
			return err
		}
	}

	return nil
}

func run(options Options) error {
	srcPath := options.SourcePath
	destPath := filepath.Join(srcPath, "{{ cookiecutter.project_slug }}")

	if srcPath == "" {
		return fmt.Errorf("source path is empty")
	}

	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		return fmt.Errorf("error checking source path: %v", err)
	}
	if !srcInfo.IsDir() {
		return fmt.Errorf("source path is not a directory: %s", srcPath)
	}

	tarData, err := createInMemoryTar(srcPath)
	if err != nil {
		return fmt.Errorf("error creating in-memory tar archive: %v", err)
	}
	slog.Debug("in-memory tar archive created")

	if err := extractInMemoryTar(tarData, destPath); err != nil {
		return fmt.Errorf("error extracting in-memory tar archive: %v", err)
	}
	slog.Debug("in-memory tar archive extracted to", "path", destPath)

	return nil
}

func printTrackedPaths() {
	for _, path := range trackedPaths {
		slog.Debug("tracked", "path", path)
	}
}

func deleteTrackedPaths() error {
	for _, path := range trackedPaths {
		slog.Debug("deleting tracked", "path", path)
		err := os.RemoveAll(path)
		if err != nil {
			slog.Error("Error deleting path", "path", path, "error", err)
			return fmt.Errorf("error deleting path: %s, %v", path, err)
		}
	}
	return nil
}
