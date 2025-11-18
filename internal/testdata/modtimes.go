// go:build ignore

// This program is used to save and apply file modification times for hardcache testing.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"
)

// save saves modification times of files under the "local" directory to "modtimes.txt".
func save() {
	f, err := os.Create("modtimes.txt")
	if err != nil {
		log.Fatal(err)
	}

	w := bufio.NewWriter(f)

	defer func() {
		if err = w.Flush(); err != nil {
			log.Fatal(err)
		}
		if err = f.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	err = filepath.Walk("local", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			log.Fatal(err)
		}

		if _, err = fmt.Fprintf(w, "%s %d\n", path, info.ModTime().UnixNano()); err != nil {
			log.Fatal(err)
		}

		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}

// apply reads modification times from "modtimes.txt" and applies them to the corresponding files.
func apply() {
	f, err := os.Open("modtimes.txt")
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	r := bufio.NewReader(f)

	for {
		var path string
		var modtime int64
		if _, err = fmt.Fscanf(r, "%s %d\n", &path, &modtime); err != nil {
			if err == io.EOF {
				return
			}

			log.Fatal(err)
		}

		t := time.Unix(0, modtime)
		if err = os.Chtimes(path, t, t); err != nil {
			log.Fatal(err)
		}
	}
}

func main() {
	saveF := flag.Bool("save", false, "save modification times")
	applyF := flag.Bool("apply", false, "apply modification times")
	flag.Parse()

	if *saveF {
		save()
	}

	if *applyF {
		apply()
	}
}
