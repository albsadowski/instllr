package main

import (
	"embed"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	cp "github.com/otiai10/copy"
	"github.com/urfave/cli/v2"
)

//go:embed templates
var templates embed.FS

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

func serviceTemplate(host string, conf *Conf, appEnv []string, targetDir string) {
	t, err := template.ParseFS(templates, "templates/service.template")
	if err != nil {
		log.Fatal(err)
	}

	data := struct {
		AppName    string
		ExecStart  string
		Env        []string
		WorkingDir string
	}{
		AppName:    host,
		ExecStart:  conf.Run,
		Env:        appEnv,
		WorkingDir: targetDir,
	}

	f, err := os.Create(fmt.Sprintf("/etc/systemd/system/%s.service", host))
	if err != nil {
		log.Fatal(err)
	}

	err = t.Execute(f, data)
	if err != nil {
		log.Fatal(err)
	}
}

func proxyTemplate(host string, port int) {
	t, err := template.ParseFS(templates, "templates/nginx.template")
	if err != nil {
		log.Fatal(err)
	}

	data := struct {
		Host string
		Port int
	}{
		Host: host,
		Port: port,
	}

	logsDir := fmt.Sprintf("/var/log/%s", host)
	if _, err := os.Stat(logsDir); err != nil {
		err = os.MkdirAll(logsDir, 0777)
		if err != nil {
			log.Fatal(err)
		}
	}

	// TODO check SSL certs

	f, err := os.Create(fmt.Sprintf("/etc/nginx/sites-enabled/%s.conf", host))
	if err != nil {
		log.Fatal(err)
	}

	err = t.Execute(f, data)
	if err != nil {
		log.Fatal(err)
	}
}

func install(
	conf *Conf,
	appEnv []string,
	src string,
	owner string,
	repo string,
	tag string,
	host string,
	port int) {
	ensureUser(host)

	targetDir := filepath.Join("/home", host, fmt.Sprintf("%s-%s-%s", owner, repo, tag))
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

	for _, step := range conf.InstallSteps {
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

	chown(targetDir, host)
	serviceTemplate(host, conf, appEnv, targetDir)
	proxyTemplate(host, port)
}

func main() {
	var owner, repo, tag, host string
	var port int
	var appEnv cli.StringSlice

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
			&cli.StringSliceFlag{
				Name:        "app-env",
				Usage:       "Application env variables",
				Destination: &appEnv,
			},
			&cli.StringFlag{
				Name:        "host",
				Usage:       "Hostname",
				Required:    true,
				Destination: &host,
			},
			&cli.IntFlag{
				Name:        "port",
				Usage:       "local application port",
				Required:    true,
				Destination: &port,
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

			install(conf, appEnv.Value(), dir, owner, repo, release.Tag, host, port)

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
