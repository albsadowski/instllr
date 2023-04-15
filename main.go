package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	cp "github.com/otiai10/copy"
	"github.com/urfave/cli/v2"
)

const appDir = "/home/albert/apps"

func loadConfig(dir string) {
	instllrJson := filepath.Join(dir, "instllr.json")
	if _, err := os.Stat(instllrJson); err != nil {
		log.Fatalf("instllr.json not found in %s, aborting", dir)
	}
}

func install(src string, owner string, repo string, tag string) {
	targetDir := filepath.Join(appDir, fmt.Sprintf("%s-%s-%s", owner, repo, tag))
	if _, err := os.Stat(targetDir); err == nil {
		log.Fatalf("directory %s already exists, aborting", targetDir)
	}

	err := os.MkdirAll(targetDir, 0777)
	if err != nil {
		log.Fatal(err)
	}

	err = cp.Copy(src, targetDir)
	if err != nil {
		log.Fatal(err)
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

			release := GetGitHubRelease(owner, repo, tag)
			if len(release.Assets) != 1 {
				log.Fatalf("Expected exactly one release asset, found %d", len(release.Assets))
			}

			dir := TmpDir()
			defer os.RemoveAll(dir)

			assetpath := FetchReleaseAsset(&release.Assets[0], dir)
			fmt.Printf("Asset: %s\n", assetpath)

			Untar(assetpath, dir)
			err := os.Remove(assetpath)
			if err != nil {
				log.Fatal(err)
			}

			loadConfig(dir)
			install(dir, owner, repo, tag)

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
