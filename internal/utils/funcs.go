package utils

import (
	"monitor/internal/types"
	"os"
	"unicode"
)

// AtomicWrite writes data to a temporary file and renames it into place.
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

// Capitalize uppercases the first rune of a string.
func Capitalize(s string) string {
	if s == "" {
		return s
	}

	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])

	return string(r)
}

// SliceDiffComparable returns values added in b and removed from a.
func SliceDiffComparable[T comparable](a, b []T) (added, removed []T) {
	aSet := types.NewSet(a...)
	bSet := types.NewSet(b...)

	added = make([]T, 0, len(b))
	removed = make([]T, 0, len(a))

	for _, aEl := range a {
		if !bSet.Exists(aEl) {
			removed = append(removed, aEl)
		}
	}

	for _, bEl := range b {
		if !aSet.Exists(bEl) {
			added = append(added, bEl)
		}
	}

	return added, removed
}
