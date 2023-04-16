package main

import (
	"archive/tar"
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xi2/xz"
)

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

		switch header.Typeflag {
		case tar.TypeDir:
			err = os.MkdirAll(filepath.Join(target, header.Name), 0777)
			if err != nil {
				log.Fatal(err)
			}
		case tar.TypeReg, tar.TypeRegA:
			fp := filepath.Join(target, header.Name)
			err = os.MkdirAll(filepath.Dir(fp), 0777)
			if err != nil {
				log.Fatal(err)
			}

			w, err := os.Create(fp)
			if err != nil {
				log.Fatal(err)
			}

			_, err = io.Copy(w, tarReader)
			if err != nil {
				log.Fatal(err)
			}
			w.Close()
		}
	}
}

func id(uname string, flag string) int {
	cmd := exec.Command("id", flag, uname)

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	id, err := strconv.Atoi(strings.TrimSpace(out.String()))
	if err != nil {
		log.Fatal(err)
	}

	return id
}

func chown(root string, uname string) {
	uid, gid := id(uname, "-u"), id(uname, "-g")
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		err = os.Chown(path, uid, gid)
		if err != nil {
			log.Fatal(err)
		}

		return nil
	})

	if err != nil {
		log.Fatal(err)
	}
}
