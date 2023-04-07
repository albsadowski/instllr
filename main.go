package main

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"
	"github.com/xi2/xz"
)

type ReleaseAsset struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
	Url  string `json:"browser_download_url"`
}

type Release struct {
	Id     int            `json:"id"`
	Tag    string         `json:"tag_name"`
	Name   string         `json:"name"`
	Assets []ReleaseAsset `json:"assets"`
}

func getRelease(owner string, repo string, tag string) *Release {
	releaseUrl := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/%s", owner, repo, tag)
	fmt.Printf("Fetching GitHub release: %s\n", releaseUrl)

	req, err := http.NewRequest(http.MethodGet, releaseUrl, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		log.Fatalf("Unexpected response from GitHub API: %d", res.StatusCode)
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

func fetchAsset(asset *ReleaseAsset, dir string) string {
	res, err := http.Get(asset.Url)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		log.Fatalf("Unexpected response from GitHub API: %d", res.StatusCode)
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

func tmpDir() string {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		log.Fatal(err)
	}
	return dir
}

func untar(path string, target string) {
	reader, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer reader.Close()

	r, err := xz.NewReader(reader, 0)
	if err != nil {
		log.Fatal(err)
	}

	tarReader := tar.NewReader(r)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatal(err)
		}

		// FIXME

		path := filepath.Join(target, header.Name)
		info := header.FileInfo()
		if info.IsDir() {
			if err = os.MkdirAll(path, info.Mode()); err != nil {
				log.Fatal(err)
			}
			continue
		}

		file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		_, err = io.Copy(file, tarReader)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func main() {
	var owner, repo, tag string

	app := &cli.App{
		Name:  "instllr",
		Usage: "install a service",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "gh-repo",
				Usage:       "GitHub repository name",
				Required:    true,
				Destination: &repo,
			},
			&cli.StringFlag{
				Name:        "gh-owner",
				Usage:       "GitHub repository owner",
				Required:    true,
				Destination: &owner,
			},
			&cli.StringFlag{
				Name:        "gh-tag",
				Usage:       "GitHub tag name",
				Destination: &tag,
			},
		},
		Action: func(*cli.Context) error {
			fmt.Printf("Installing %s/%s:%s\n", owner, repo, tag)

			release := getRelease(owner, repo, tag)
			if len(release.Assets) != 1 {
				log.Fatalf("Expected exactly one release asset, found %d", len(release.Assets))
			}

			dir := tmpDir()
			defer os.RemoveAll(dir)

			assetpath := fetchAsset(&release.Assets[0], dir)
			fmt.Printf("Asset: %s\n", assetpath)

			untar(assetpath, dir)

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
