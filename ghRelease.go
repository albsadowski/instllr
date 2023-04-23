package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

type ReleaseAsset struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
	Url  string `json:"url"`
}

type Release struct {
	Id     int            `json:"id"`
	Tag    string         `json:"tag_name"`
	Name   string         `json:"name"`
	Assets []ReleaseAsset `json:"assets"`
}

func getGitHubRelease(s *Service) *Release {
	var tagFragment string
	if s.Tag == "" || s.Tag == "latest" {
		tagFragment = "latest"
	} else {
		tagFragment = fmt.Sprintf("tags/%s", s.Tag)
	}

	releaseUrl := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/%s", s.Owner, s.Repo, tagFragment)
	fmt.Printf("Fetching GitHub release: %s\n", releaseUrl)

	req := unsafeGet(http.NewRequest(http.MethodGet, releaseUrl, nil))
	ghToken := os.Getenv("GH_TOKEN")
	if ghToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ghToken))
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	res := unsafeGet(http.DefaultClient.Do(req))
	defer res.Body.Close()

	if res.StatusCode != 200 {
		log.Fatalf("unexpected response from GitHub API (%s): %d", req.URL, res.StatusCode)
	}

	resBody := unsafeGet(ioutil.ReadAll(res.Body))

	var release Release
	unsafe(json.Unmarshal(resBody, &release))

	return &release
}

func fetchReleaseAsset(asset *ReleaseAsset, dir string) string {
	req := unsafeGet(http.NewRequest(http.MethodGet, asset.Url, nil))
	ghToken := os.Getenv("GH_TOKEN")
	if ghToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ghToken))
	}
	req.Header.Set("Accept", "application/octet-stream")

	res := unsafeGet(http.DefaultClient.Do(req))
	defer res.Body.Close()

	if res.StatusCode != 200 {
		log.Fatalf("unexpected response from GitHub API (%s): %d", asset.Url, res.StatusCode)
	}

	assetpath := filepath.Join(dir, asset.Name)
	out := unsafeGet(os.Create(assetpath))
	defer out.Close()

	unsafeGet(io.Copy(out, res.Body))

	return assetpath
}
