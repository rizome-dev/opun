package plugin

// Copyright (C) 2025 Rizome Labs, Inc.
//
// This program is free software; you can redistribute it and/or
// modify it under the terms of the GNU General Public License
// as published by the Free Software Foundation; either version 2
// of the License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program; if not, write to the Free Software
// Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301, USA.

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rizome-dev/opun/internal/utils"
	pluginTypes "github.com/rizome-dev/opun/pkg/plugin"
	"gopkg.in/yaml.v3"
)

// PluginFetcher handles downloading and installing plugins from remote repositories
type PluginFetcher struct {
	manager              *Manager
	pluginsDir           string
	client               *http.Client
	githubFallbackBranch string
	githubFallbackURL    string
}

// NewPluginFetcher creates a new plugin fetcher
func NewPluginFetcher(manager *Manager, pluginsDir string) *PluginFetcher {
	return &PluginFetcher{
		manager:    manager,
		pluginsDir: pluginsDir,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchAndInstall downloads a plugin from a remote URL and installs it
func (f *PluginFetcher) FetchAndInstall(ctx context.Context, repoURL string) error {
	// Parse and validate URL
	parsed, err := url.Parse(repoURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Determine source type
	var downloadURL string
	var repoName string

	switch {
	case strings.Contains(parsed.Host, "github.com"):
		downloadURL, repoName, err = f.getGitHubDownloadURL(parsed)
	case strings.Contains(parsed.Host, "gitlab.com"):
		downloadURL, repoName, err = f.getGitLabDownloadURL(parsed)
	default:
		return fmt.Errorf("unsupported repository host: %s", parsed.Host)
	}

	if err != nil {
		return fmt.Errorf("failed to get download URL: %w", err)
	}

	// Create temp directory for download
	tempDir, err := os.MkdirTemp("", "opun-plugin-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Download archive
	archivePath := filepath.Join(tempDir, "plugin-archive")
	if err := f.downloadFile(ctx, downloadURL, archivePath); err != nil {
		return fmt.Errorf("failed to download plugin: %w", err)
	}

	// Extract archive
	extractDir := filepath.Join(tempDir, "extracted")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return fmt.Errorf("failed to create extraction directory: %w", err)
	}

	if err := f.extractArchive(archivePath, extractDir); err != nil {
		return fmt.Errorf("failed to extract archive: %w", err)
	}

	// Find plugin manifest
	manifestPath, err := f.findPluginManifest(extractDir)
	if err != nil {
		return fmt.Errorf("failed to find plugin manifest: %w", err)
	}

	// Read and validate manifest
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest pluginTypes.PluginManifest
	if err := yaml.Unmarshal(manifestData, &manifest); err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Use repo name if plugin name not specified
	if manifest.Name == "" {
		manifest.Name = repoName
	}

	// Prepare plugin directory
	pluginDir := filepath.Join(f.pluginsDir, manifest.Name)

	// Check if plugin already exists
	if _, err := os.Stat(pluginDir); err == nil {
		return fmt.Errorf("plugin '%s' already exists. Use 'opun update plugin %s' to update", manifest.Name, manifest.Name)
	}

	// Create plugin directory with proper ownership
	if err := utils.EnsureDir(pluginDir); err != nil {
		return fmt.Errorf("failed to create plugin directory: %w", err)
	}

	// Copy plugin files
	sourceDir := filepath.Dir(manifestPath)
	if err := f.copyDirectory(sourceDir, pluginDir); err != nil {
		os.RemoveAll(pluginDir) // Clean up on failure
		return fmt.Errorf("failed to copy plugin files: %w", err)
	}

	// Initialize plugin in manager
	if err := f.manager.LoadPlugin(manifest.Name); err != nil {
		os.RemoveAll(pluginDir) // Clean up on failure
		return fmt.Errorf("failed to load plugin: %w", err)
	}

	return nil
}

// getGitHubDownloadURL converts a GitHub repo URL to archive download URL
func (f *PluginFetcher) getGitHubDownloadURL(parsed *url.URL) (string, string, error) {
	// Extract owner/repo from path
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid GitHub URL format")
	}

	owner := parts[0]
	repo := strings.TrimSuffix(parts[1], ".git")

	// Check if specific branch/tag is specified
	branch := "main"
	if len(parts) > 3 && parts[2] == "tree" {
		branch = parts[3]
	}

	// GitHub archive URL format
	downloadURL := fmt.Sprintf("https://github.com/%s/%s/archive/refs/heads/%s.zip", owner, repo, branch)

	// Try main branch first, fall back to master
	if branch == "main" {
		// We'll handle fallback in downloadFile if needed
		f.githubFallbackBranch = "master"
		f.githubFallbackURL = fmt.Sprintf("https://github.com/%s/%s/archive/refs/heads/master.zip", owner, repo)
	}

	return downloadURL, repo, nil
}

// getGitLabDownloadURL converts a GitLab repo URL to archive download URL
func (f *PluginFetcher) getGitLabDownloadURL(parsed *url.URL) (string, string, error) {
	// Extract namespace/project from path
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid GitLab URL format")
	}

	// GitLab can have nested namespaces
	project := strings.TrimSuffix(parts[len(parts)-1], ".git")
	namespace := strings.Join(parts[:len(parts)-1], "/")

	// Check if specific branch/tag is specified
	branch := "main"

	// GitLab archive URL format
	downloadURL := fmt.Sprintf("https://gitlab.com/%s/%s/-/archive/%s/%s-%s.tar.gz",
		namespace, project, branch, project, branch)

	return downloadURL, project, nil
}

// downloadFile downloads a file from URL to destination
func (f *PluginFetcher) downloadFile(ctx context.Context, url string, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Handle GitHub fallback for main->master
	if resp.StatusCode == http.StatusNotFound && f.githubFallbackURL != "" {
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, f.githubFallbackURL, nil)
		if err != nil {
			return err
		}
		resp, err = f.client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	// Create destination file
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	// Copy response body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

// extractArchive extracts a zip or tar.gz archive
func (f *PluginFetcher) extractArchive(archivePath, destDir string) error {
	// Try zip first
	if err := f.extractZip(archivePath, destDir); err == nil {
		return nil
	}

	// Try tar.gz
	if err := f.extractTarGz(archivePath, destDir); err == nil {
		return nil
	}

	return fmt.Errorf("unable to extract archive - unsupported format")
}

// extractZip extracts a zip archive
func (f *PluginFetcher) extractZip(archivePath, destDir string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, file := range r.File {
		if err := f.extractZipFile(file, destDir); err != nil {
			return err
		}
	}

	return nil
}

// extractZipFile extracts a single file from zip archive
func (f *PluginFetcher) extractZipFile(file *zip.File, destDir string) error {
	// Clean the path
	path := filepath.Join(destDir, file.Name)

	// Prevent directory traversal
	if !strings.HasPrefix(path, filepath.Clean(destDir)+string(os.PathSeparator)) {
		return fmt.Errorf("invalid file path: %s", file.Name)
	}

	if file.FileInfo().IsDir() {
		return os.MkdirAll(path, file.Mode())
	}

	// Create directory for file
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	// Open file in archive
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	// Create destination file
	out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, rc)
	return err
}

// extractTarGz extracts a tar.gz archive
func (f *PluginFetcher) extractTarGz(archivePath, destDir string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Clean the path
		path := filepath.Join(destDir, header.Name)

		// Prevent directory traversal
		if !strings.HasPrefix(path, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			// Safely convert int64 to uint32 for file mode
			// Ensure the value fits in uint32 range
			mode := os.FileMode(uint32(header.Mode) & 0777)
			if err := f.extractTarFile(tr, path, mode); err != nil {
				return err
			}
		}
	}

	return nil
}

// extractTarFile extracts a single file from tar archive
func (f *PluginFetcher) extractTarFile(r io.Reader, destPath string, mode os.FileMode) error {
	// Create directory for file
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	// Create destination file
	out, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, r)
	return err
}

// findPluginManifest searches for plugin.yaml or opun-plugin.yaml
func (f *PluginFetcher) findPluginManifest(dir string) (string, error) {
	manifestNames := []string{"plugin.yaml", "plugin.yml", "opun-plugin.yaml", "opun-plugin.yml"}

	var foundPath string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		for _, name := range manifestNames {
			if strings.ToLower(filepath.Base(path)) == name {
				// Prefer manifest in root or first-level subdirectory
				relPath, _ := filepath.Rel(dir, path)
				depth := len(strings.Split(relPath, string(os.PathSeparator)))

				if foundPath == "" || depth < 3 {
					foundPath = path
					if depth == 1 {
						return filepath.SkipAll // Found in root, stop searching
					}
				}
			}
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	if foundPath == "" {
		return "", fmt.Errorf("no plugin manifest found (looked for: %s)", strings.Join(manifestNames, ", "))
	}

	return foundPath, nil
}

// copyDirectory recursively copies a directory
func (f *PluginFetcher) copyDirectory(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate destination path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return utils.EnsureDir(dstPath)
		}

		// Copy file
		return f.copyFile(path, dstPath, info.Mode())
	})
}

// copyFile copies a single file
func (f *PluginFetcher) copyFile(src, dst string, mode os.FileMode) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}
