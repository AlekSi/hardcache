package cache

import (
	"slices"
	"testing"
	"time"
)

func TestSortForMaxSize(t *testing.T) {
	t.Parallel()

	files := []fileInfo{
		{Name: "newer", LastUse: time.Unix(2, 0)},
		{Name: "older", LastUse: time.Unix(1, 0)},
	}

	t.Run("cutoff only", func(t *testing.T) {
		actual := slices.Clone(files)
		if sortForMaxSize(actual, 2, nil) {
			t.Fatal("sortForMaxSize unexpectedly sorted without a maximum size")
		}
		if !slices.Equal(actual, files) {
			t.Fatalf("sortForMaxSize changed files: got %v, want %v", actual, files)
		}
	})

	t.Run("cutoff is sufficient", func(t *testing.T) {
		actual := slices.Clone(files)
		maxSize := int64(2)
		if sortForMaxSize(actual, 2, &maxSize) {
			t.Fatal("sortForMaxSize unexpectedly sorted at the maximum size")
		}
		if !slices.Equal(actual, files) {
			t.Fatalf("sortForMaxSize changed files: got %v, want %v", actual, files)
		}
	})

	t.Run("maximum size exceeded", func(t *testing.T) {
		actual := slices.Clone(files)
		maxSize := int64(1)
		if !sortForMaxSize(actual, 2, &maxSize) {
			t.Fatal("sortForMaxSize did not sort above the maximum size")
		}
		if actual[0].Name != "older" || actual[1].Name != "newer" {
			t.Fatalf("sortForMaxSize sorted files incorrectly: %v", actual)
		}
	})
}
