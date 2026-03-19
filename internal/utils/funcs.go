package utils

import (
	"os"
	"unicode"
)

func AtomicWrite(
	path, tmpPath string,
	data []byte,
	perm os.FileMode,
) error {
	err := os.WriteFile(tmpPath, data, perm)

	if err != nil {
		return err
	}

	return os.Rename(tmpPath, path)
}

func Capitalize(s string) string {
	if s == "" {
		return s
	}

	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])

	return string(r)
}
