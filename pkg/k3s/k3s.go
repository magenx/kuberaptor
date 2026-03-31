// Kuberaptor
// Copyright (c) 2026 Kuberaptor (https://kuberaptor.com)
// SPDX-License-Identifier: MIT

package k3s

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	githubReleasesURL = "https://api.github.com/repos/k3s-io/k3s/tags"
	cacheDir          = ".kuberaptor"
	cacheFile         = "k3s-releases.yaml"
	cacheExpiry       = 7 * 24 * time.Hour
)

// Release represents a k3s release
type Release struct {
	Name string `json:"name"`
}

// GetAvailableReleases returns all available k3s releases
func GetAvailableReleases(ctx context.Context) ([]string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	cacheDirPath := filepath.Join(homeDir, cacheDir)
	cacheFilePath := filepath.Join(cacheDirPath, cacheFile)

	if err := os.MkdirAll(cacheDirPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	if fileInfo, err := os.Stat(cacheFilePath); err == nil {
		age := time.Since(fileInfo.ModTime())
		if age <= cacheExpiry {
			data, err := os.ReadFile(cacheFilePath)
			if err == nil {
				var releases []string
				if err := yaml.Unmarshal(data, &releases); err == nil {
					return releases, nil
				}
			}
		}
		os.Remove(cacheFilePath)
	}

	releases, err := fetchReleasesFromGitHub(ctx)
	if err != nil {
		return nil, err
	}

	data, err := yaml.Marshal(releases)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal releases: %w", err)
	}

	if err := os.WriteFile(cacheFilePath, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to cache releases: %v\n", err)
	}

	return releases, nil
}

func fetchReleasesFromGitHub(ctx context.Context) ([]string, error) {
	var allReleases []string
	page := 1
	perPage := 100

	client := &http.Client{Timeout: 30 * time.Second}

	for {
		url := fmt.Sprintf("%s?per_page=%d&page=%d", githubReleasesURL, perPage, page)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch releases: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
		}

		var releases []Release
		if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		if len(releases) == 0 {
			break
		}

		for _, release := range releases {
			allReleases = append(allReleases, release.Name)
		}

		linkHeader := resp.Header.Get("Link")
		if !strings.Contains(linkHeader, "rel=\"next\"") {
			break
		}

		page++
	}

	for i, j := 0, len(allReleases)-1; i < j; i, j = i+1, j-1 {
		allReleases[i], allReleases[j] = allReleases[j], allReleases[i]
	}

	return allReleases, nil
}

func GenerateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}
