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
	"strings"
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

func (r *Release) GetAsset(assetName string) *ReleaseAsset {
	if assetName == "" {
		if len(r.Assets) == 0 {
			return nil
		}
		return &r.Assets[0]
	}

	for _, asset := range r.Assets {
		if strings.HasPrefix(asset.Name, assetName) {
			return &asset
		}
	}

	return nil
}

func ghReq(cfg *InstllrConf, method string, url string) *http.Request {
	req := unsafeGet(http.NewRequest(method, url, nil))

	if cfg.GhToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", cfg.GhToken))
	}

	return req
}

func getGitHubRelease(cfg *InstllrConf, s *Service) *Release {
	var tagFragment string
	if s.Tag == "" || s.Tag == "latest" {
		tagFragment = "latest"
	} else {
		tagFragment = fmt.Sprintf("tags/%s", s.Tag)
	}

	releaseUrl := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/%s", s.Owner, s.Repo, tagFragment)
	fmt.Printf("Fetching GitHub release: %s\n", releaseUrl)

	req := ghReq(cfg, http.MethodGet, releaseUrl)
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

func fetchReleaseAsset(cfg *InstllrConf, asset *ReleaseAsset, dir string) string {
	req := ghReq(cfg, http.MethodGet, asset.Url)
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
