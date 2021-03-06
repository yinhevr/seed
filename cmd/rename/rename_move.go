package main

import (
	"github.com/glvd/seed"
	"github.com/godcong/go-trait"
	"os"
	"path/filepath"
	"strings"
)

var log1 = trait.NewZapSugar()

func main() {
	args := os.Args
	dir, err := os.Getwd()
	if len(args) > 1 {
		err = nil
		dir = args[1]
	}
	if err != nil {
		log1.Info("wd:", err)
		return
	}
	files := seed.GetFiles(dir)

	for _, f := range files {
		dir, name := filepath.Split(f)
		dirname := strings.Split(dir, string(os.PathSeparator))
		log1.With("role", getRealName(dirname)).Info("proc")
		ext := filepath.Ext(name)
		nDir := getDir2(dir)
		_ = os.MkdirAll(nDir, os.ModePerm)
		nPath := filepath.Join(nDir, getRealName(dirname)+ext)
		e := os.Rename(f, nPath)
		log1.Error(e)
	}
}

func getDir2(d string) string {
	args := os.Args
	dir, err := os.Getwd()
	if len(args) > 2 {
		err = nil
		dir = args[2]
	}
	if err != nil {
		return d
	}
	return dir
}

func getRealName(s []string) string {
	size := len(s)
	if s == nil || size == 0 {
		return ""
	}
	for last := size - 1; last > 0; last-- {
		if s[last] != "" {
			return s[last]
		}
	}
	return ""
}
