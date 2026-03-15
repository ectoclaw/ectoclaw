package upgrade

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"testing"
)

func TestAssetName(t *testing.T) {
	name, err := assetName()
	if err != nil {
		t.Fatalf("assetName() error: %v", err)
	}
	if name == "" {
		t.Fatal("assetName() returned empty string")
	}
}

func TestExtractFromTarGz(t *testing.T) {
	want := []byte("fake binary content")

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	hdr := &tar.Header{
		Name:     "ectoclaw",
		Typeflag: tar.TypeReg,
		Size:     int64(len(want)),
		Mode:     0755,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(want); err != nil {
		t.Fatal(err)
	}
	tw.Close()
	gz.Close()

	got, err := extractFromTarGz(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("extractFromTarGz error: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestExtractFromTarGz_NotFound(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(&buf)
	tw.Close()
	gz.Close()

	_, err := extractFromTarGz(bytes.NewReader(buf.Bytes()))
	if err == nil {
		t.Fatal("expected error for missing binary, got nil")
	}
}

func TestFindAsset(t *testing.T) {
	release := &githubRelease{
		TagName: "v1.0.0",
		Assets: []githubAsset{
			{Name: "ectoclaw_Darwin_arm64.tar.gz", BrowserDownloadURL: "https://example.com/a"},
			{Name: "ectoclaw_Linux_x86_64.tar.gz", BrowserDownloadURL: "https://example.com/b"},
		},
	}

	asset, err := findAsset(release, "ectoclaw_Linux_x86_64.tar.gz")
	if err != nil {
		t.Fatalf("findAsset error: %v", err)
	}
	if asset.BrowserDownloadURL != "https://example.com/b" {
		t.Fatalf("unexpected URL: %s", asset.BrowserDownloadURL)
	}

	_, err = findAsset(release, "ectoclaw_Windows_x86_64.zip")
	if err == nil {
		t.Fatal("expected error for missing asset")
	}
}
