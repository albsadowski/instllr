package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	cp "github.com/otiai10/copy"
	"github.com/urfave/cli/v2"
)

func ensureUser(appName string) {
	err := exec.Command("id", "-u", appName).Run()
	if err == nil {
		fmt.Printf("User '%s' already exists\n", appName)
		return
	}

	cmd := exec.Command("useradd", "-mrU", appName)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func install(
	appName string,
	steps []string,
	src string,
	owner string,
	repo string,
	tag string) {
	ensureUser(appName)

	targetDir := filepath.Join("/home", appName, fmt.Sprintf("%s-%s-%s", owner, repo, tag))
	if _, err := os.Stat(targetDir); err == nil {
		log.Fatalf("directory %s already exists, aborting", targetDir)
	}

	err := os.MkdirAll(targetDir, 0777)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Installing at the target directory: %s\n", targetDir)
	err = cp.Copy(src, targetDir)
	if err != nil {
		log.Fatal(err)
	}

	for _, step := range steps {
		split := strings.Fields(step)
		cmd := exec.Command(split[0], split[1:]...)
		cmd.Dir = targetDir

		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err != nil {
			log.Fatalf("install step '%s' failed, aborting", step)
		}
	}

	chown(targetDir, appName)
}

func main() {
	var appName, owner, repo, tag string

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
			&cli.StringFlag{
				Name:        "app-name",
				Usage:       "Application name",
				Required:    true,
				Destination: &appName,
			},
		},
		Action: func(*cli.Context) error {
			fmt.Printf("Installing %s/%s:%s\n", owner, repo, tag)

			release := getGitHubRelease(owner, repo, tag)
			if len(release.Assets) != 1 {
				log.Fatalf("Expected exactly one release asset, found %d", len(release.Assets))
			}

			dir := tmpDir()
			defer os.RemoveAll(dir)

			assetpath := fetchReleaseAsset(&release.Assets[0], dir)
			fmt.Printf("Asset: %s\n", assetpath)

			untar(assetpath, dir)
			err := os.Remove(assetpath)
			if err != nil {
				log.Fatal(err)
			}

			conf := loadConfig(dir)
			checkDeps(conf.Require)

			install(appName, conf.InstallSteps, dir, owner, repo, release.Tag)

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
