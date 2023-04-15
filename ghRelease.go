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

func GetGitHubRelease(owner string, repo string, tag string) *Release {
	var tagFragment string
	if tag == "latest" {
		tagFragment = tag
	} else {
		tagFragment = fmt.Sprintf("tags/%s", tag)
	}

	releaseUrl := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/%s", owner, repo, tagFragment)
	fmt.Printf("Fetching GitHub release: %s\n", releaseUrl)

	req, err := http.NewRequest(http.MethodGet, releaseUrl, nil)
	if err != nil {
		log.Fatal(err)
	}

	ghToken := os.Getenv("GH_TOKEN")
	if ghToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ghToken))
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		log.Fatalf("unexpected response from GitHub API (%s): %d", req.URL, res.StatusCode)
	}

	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	var release Release
	err = json.Unmarshal(resBody, &release)
	if err != nil {
		log.Fatal(err)
	}

	return &release
}

func FetchReleaseAsset(asset *ReleaseAsset, dir string) string {
	req, err := http.NewRequest(http.MethodGet, asset.Url, nil)
	if err != nil {
		log.Fatal(err)
	}

	ghToken := os.Getenv("GH_TOKEN")
	if ghToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ghToken))
	}
	req.Header.Set("Accept", "application/octet-stream")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		log.Fatalf("unexpected response from GitHub API (%s): %d", asset.Url, res.StatusCode)
	}

	assetpath := filepath.Join(dir, asset.Name)
	out, err := os.Create(assetpath)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	_, err = io.Copy(out, res.Body)
	if err != nil {
		log.Fatal(err)
	}

	return assetpath
}
