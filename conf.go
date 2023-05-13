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
	App        string   `json:"app"`
	Version    []string `json:"version"`
	MinVersion string   `json:"minVersion"`
}

type EnvConf struct {
	Require []string `json:"require"`
}

type Conf struct {
	Require     []ConfRequire `json:"require"`
	InstallStep []string      `json:"install"`
	Run         []string      `json:"run"`
	Env         EnvConf       `json:"env"`
}

type InstllrConf struct {
	GhToken string `json:"ghToken"`
}

func command(cmds ...string) *exec.Cmd {
	cmd := exec.Command(cmds[0], cmds[1:]...)

	nodePath := os.Getenv("NODE_PATH")
	if nodePath != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("PATH=%s:$PATH", nodePath))
	}

	return cmd
}

func loadInstllrConfig() *InstllrConf {
	cfgPath := filepath.Join(unsafeGet(os.UserHomeDir()), ".instllr.json")
	_, err := os.Stat(cfgPath)

	if err == nil {
		content := unsafeGet(ioutil.ReadFile(cfgPath))
		var cfg InstllrConf
		unsafe(json.Unmarshal(content, &cfg))
		return &cfg
	}

	return &InstllrConf{GhToken: os.Getenv("GH_TOKEN")}
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

	if len(conf.Run) == 0 {
		log.Fatal("invalid installr.json")
	}

	for _, r := range conf.Require {
		if r.App == "" {
			log.Fatal("invalid instllr.json")
		}
	}

	return &conf
}

func assertExists(executable string) string {
	cmd := command("which", executable)

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		log.Fatalf("Dependency check: failed to check dependency: %s", executable)
	}

	path := strings.TrimSpace(out.String())
	fmt.Printf("Dependency check: %s found at %s", executable, path)

	return path
}

func getVersion(path string, conf ConfRequire) string {
	if len(conf.Version) == 0 {
		log.Fatalf("no version command specified for %s", conf.App)
	}

	if conf.Version[0] != conf.App {
		log.Fatalf("invalid dependency spec for %s", conf.App)
	}

	args := append([]string{path}, conf.Version[1:]...)
	cmd := command(args...)

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		log.Fatalf("dependency check: failed to check version '%s': %s", path, err)
	}

	return strings.TrimSpace(out.String())
}

var vreplacer = strings.NewReplacer(
	"v", "",
	"\n", "",
	"\r", "",
)

func cmpVersion(v1 string, v2 string) int {
	v1clean := strings.Split(vreplacer.Replace(v1), ".")
	v2clean := strings.Split(vreplacer.Replace(v2), ".")

	if len(v1clean) != len(v2clean) {
		log.Fatalf("incompatible version format: %s vs %s", v1, v2)
	}

	for ix, v := range v1clean {
		if v < v2clean[ix] {
			return -1
		}

		if v > v2clean[ix] {
			break
		}
	}

	for ix, v := range v1clean {
		if v != v2clean[ix] {
			return 1
		}
	}

	return 0
}

func assertVersion(ver string, minv string) {
	if cmpVersion(ver, minv) < 0 {
		log.Fatalf("min version required: %s, found: %s", minv, ver)
	}
}

func resolveDeps(require []ConfRequire) *map[string]string {
	deps := make(map[string]string)

	for _, r := range require {
		path := assertExists(r.App)
		version := getVersion(path, r)
		fmt.Printf("Dependency check: %s version: %s\n", r.App, version)

		assertVersion(version, r.MinVersion)
		fmt.Printf("Dependency check: %s OK\n", r.App)

		deps[r.App] = path
	}

	return &deps
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
