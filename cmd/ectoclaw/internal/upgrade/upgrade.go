package upgrade

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	githubAPIBase = "https://api.github.com/repos/ectoclaw/ectoclaw/releases"
	userAgent     = "ectoclaw-upgrade"
)

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func fetchRelease(nightly bool) (*githubRelease, error) {
	var url string
	if nightly {
		url = githubAPIBase + "/tags/nightly"
	} else {
		url = githubAPIBase + "/latest"
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &release, nil
}

func assetName() (string, error) {
	type platform struct {
		os   string
		arch string
	}

	osMap := map[string]string{
		"darwin":  "Darwin",
		"linux":   "Linux",
		"windows": "Windows",
	}
	archMap := map[string]string{
		"amd64": "x86_64",
		"arm64": "arm64",
	}

	osName, ok := osMap[runtime.GOOS]
	if !ok {
		return "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
	archName, ok := archMap[runtime.GOARCH]
	if !ok {
		return "", fmt.Errorf("unsupported architecture: %s", runtime.GOARCH)
	}

	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}

	return fmt.Sprintf("ectoclaw_%s_%s%s", osName, archName, ext), nil
}

func findAsset(release *githubRelease, name string) (*githubAsset, error) {
	for i := range release.Assets {
		if release.Assets[i].Name == name {
			return &release.Assets[i], nil
		}
	}
	return nil, fmt.Errorf("asset %q not found in release %s", name, release.TagName)
}

func downloadAndReplace(url, binaryPath string) error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf(
			"automatic upgrade is not supported on Windows because the running executable is locked\n" +
				"Please download the latest release manually from https://github.com/ectoclaw/ectoclaw/releases",
		)
	}

	fmt.Printf("Downloading %s...\n", filepath.Base(url))

	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned %d", resp.StatusCode)
	}

	archiveBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read archive: %w", err)
	}

	fmt.Println("Extracting...")

	ext := ".tar.gz"
	if strings.HasSuffix(url, ".zip") {
		ext = ".zip"
	}

	binary, err := extractBinary(bytes.NewReader(archiveBytes), ext)
	if err != nil {
		return fmt.Errorf("extract: %w", err)
	}

	// Write to temp file in same directory for atomic rename
	dir := filepath.Dir(binaryPath)
	tmp, err := os.CreateTemp(dir, "ectoclaw-upgrade-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		tmp.Close()
		os.Remove(tmpPath) // no-op if rename succeeded
	}()

	if _, err := tmp.Write(binary); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	// Match current binary permissions
	info, err := os.Stat(binaryPath)
	if err != nil {
		return fmt.Errorf("stat binary: %w", err)
	}
	if err := os.Chmod(tmpPath, info.Mode()); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}

	if err := os.Rename(tmpPath, binaryPath); err != nil {
		return fmt.Errorf("replace binary: %w", err)
	}
	return nil
}

func extractBinary(r io.Reader, ext string) ([]byte, error) {
	switch ext {
	case ".tar.gz":
		return extractFromTarGz(r)
	case ".zip":
		// zip.NewReader requires io.ReaderAt + size; buffer first
		data, err := io.ReadAll(r)
		if err != nil {
			return nil, err
		}
		return extractFromZip(data)
	default:
		return nil, fmt.Errorf("unsupported archive format: %s", ext)
	}
}

func extractFromTarGz(r io.Reader) ([]byte, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("tar: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		base := filepath.Base(hdr.Name)
		if base == "ectoclaw" || base == "ectoclaw.exe" {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("ectoclaw binary not found in archive")
}

func extractFromZip(data []byte) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("zip: %w", err)
	}
	for _, f := range zr.File {
		base := filepath.Base(f.Name)
		if base == "ectoclaw" || base == "ectoclaw.exe" {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("ectoclaw binary not found in archive")
}

// currentBinaryPath returns the resolved path of the running executable.
func currentBinaryPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locate executable: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return "", fmt.Errorf("resolve symlinks: %w", err)
	}
	return resolved, nil
}
