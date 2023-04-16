package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type ConfRequire struct {
	App        string `json:"app"`
	Version    string `json:"version"`
	MinVersion string `json:"minVersion"`
}

type EnvConf struct {
	Require []string `json:"require"`
}

type Conf struct {
	Require      []ConfRequire `json:"require"`
	InstallSteps []string      `json:"install"`
	Run          string        `json:"run"`
	Env          EnvConf       `json:"env"`
}

func loadConfig(dir string) *Conf {
	instllrJson := filepath.Join(dir, "instllr.json")
	if _, err := os.Stat(instllrJson); err != nil {
		log.Fatalf("instllr.json not found in %s, aborting", dir)
	}

	content, err := ioutil.ReadFile(instllrJson)
	if err != nil {
		log.Fatal("failed to open instllr.json", err)
	}

	var conf Conf
	err = json.Unmarshal(content, &conf)
	if err != nil {
		log.Fatal("failed to unmarshal instllr.json", err)
	}

	if conf.Run == "" {
		log.Fatal("invalid installr.json")
	}

	for _, r := range conf.Require {
		if r.App == "" {
			log.Fatal("invalid instllr.json")
		}
	}

	return &conf
}

func assertExists(executable string) {
	cmd := exec.Command("which", executable)

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		log.Fatalf("Dependency check: failed to check dependency: %s", executable)
	}

	fmt.Printf("Dependency check: %s found at %s", executable, out.String())
}

func getVersion(vcmd string) string {
	split := strings.Fields(vcmd)
	cmd := exec.Command(split[0], split[1:]...)

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		log.Fatalf("Dependency check: failed to execute '%s': %s", vcmd, err)
	}

	return strings.TrimSpace(out.String())
}

var vreplacer = strings.NewReplacer(
	"v", "",
	"\n", "",
	"\r", "",
)

func assertVersion(ver string, minv string) {
	vclean := strings.Split(vreplacer.Replace(ver), ".")
	minvclean := strings.Split(vreplacer.Replace(minv), ".")

	if len(vclean) != len(minvclean) {
		log.Fatalf("incompatible version format: %s vs %s", ver, minv)
	}

	fatal := func() {
		log.Fatalf("min version required: %s, found: %s", minv, ver)
	}

	for ix, v := range vclean {
		if v < minvclean[ix] {
			fatal()
		}

		if v > minvclean[ix] {
			break
		}
	}
}

func checkDeps(require []ConfRequire) {
	for _, r := range require {
		assertExists(r.App)
		version := getVersion(r.Version)
		fmt.Printf("Dependency check: %s version: %s\n", r.App, version)

		assertVersion(version, r.MinVersion)
		fmt.Printf("Dependency check: %s OK\n", r.App)
	}
}

func checkEnv(conf EnvConf, appEnv []string) {
	for _, r := range conf.Require {
		found := false
		for _, e := range appEnv {
			if strings.HasPrefix(e, fmt.Sprintf("%s=", r)) {
				found = true
				break
			}
		}
		if !found {
			log.Fatalf("required application env variable '%s' not found", r)
		}
	}
}
