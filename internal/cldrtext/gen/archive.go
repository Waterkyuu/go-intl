package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"time"
)

// version identifies the Unicode release shared by all CLDR inputs.
const version = "48.2.0"

// digests pins the exact upstream archives used to generate the tables.
var digests = map[string]string{
	"cldr-core":         "5310e0c7a06c1feb83dc8e54c8584bbe0b9c2ea8320172a18d541986b124585d",
	"cldr-dates-full":   "0256f1cefeca14f7d515be4872dda48fcdd7e75381430f25aafd0143eae5b430",
	"cldr-misc-full":    "c6ba8384d7ea8701cf86935db0461379231ddd970cc41c249f4a33b9857ded3c",
	"cldr-numbers-full": "2d17a1453c559a62112caeed52e0bcfe3cb8539c99d239ae7b7ed4d0827679d9",
}

// download verifies the checksum before any archive content is decoded.
func download(pkg string) ([]byte, error) {
	url := fmt.Sprintf("https://registry.npmjs.org/%s/-/%s-%s.tgz", pkg, pkg, version)
	client := http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: %s", pkg, resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if fmt.Sprintf("%x", sha256.Sum256(data)) != digests[pkg] {
		return nil, fmt.Errorf("unexpected SHA-256 digest for %s", pkg)
	}
	return data, nil
}

// readArchive decodes matching JSON entries and verifies the upstream data license.
func readArchive(pkg, filename string, license []byte, decode func(io.Reader) error) error {
	data, err := download(pkg)
	if err != nil {
		return err
	}
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer func() { _ = gz.Close() }()
	tr := tar.NewReader(gz)
	var matched, licensed bool
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		switch {
		case header.Name == "package/LICENSE":
			body, err := io.ReadAll(tr)
			if err != nil {
				return err
			}
			if !bytes.Equal(body, license) {
				return fmt.Errorf("review the updated %s license before regenerating", pkg)
			}
			licensed = true
		case path.Base(header.Name) == filename:
			if err := decode(tr); err != nil {
				return fmt.Errorf("%s/%s: %w", pkg, header.Name, err)
			}
			matched = true
		}
	}
	if !matched || !licensed {
		return fmt.Errorf("%s must contain %s and the data license", pkg, filename)
	}
	return nil
}

// loadLocales collects one typed document per locale from a pinned archive.
func loadLocales[T any](pkg, filename string, license []byte) (map[string]T, error) {
	locales := make(map[string]T)
	err := readArchive(pkg, filename, license, func(r io.Reader) error {
		var doc struct {
			// Main associates the file's locale with its CLDR payload.
			Main map[string]T
		}
		if err := json.NewDecoder(r).Decode(&doc); err != nil {
			return err
		}
		if len(doc.Main) != 1 {
			return fmt.Errorf("expected one locale in main")
		}
		for tag, data := range doc.Main {
			if _, exists := locales[tag]; exists {
				return fmt.Errorf("duplicate locale %q", tag)
			}
			locales[tag] = data
		}
		return nil
	})
	return locales, err
}
