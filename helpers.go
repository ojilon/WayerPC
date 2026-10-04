package main

import (
	"os"
	"path/filepath"
	"strings"
)

func base(p string) string {
	b := filepath.Base(p)
	if b == "" {
		return p
	}
	return b
}

func isDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func sameDir(a, b string) bool {
	aa, errA := filepath.Abs(a)
	bb, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return strings.EqualFold(a, b)
	}
	return strings.EqualFold(aa, bb)
}
