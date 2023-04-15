package main

import (
	"archive/tar"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/xi2/xz"
)

func TmpDir() string {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		log.Fatal(err)
	}
	return dir
}

func Untar(path string, target string) {
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
