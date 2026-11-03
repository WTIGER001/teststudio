package checks

import (
	"os"
	"testing"
)

func init() {
	if err := os.WriteFile("/output/package-initialized", []byte("initialized"), 0600); err != nil {
		panic(err)
	}
}

func TestTotal(t *testing.T) {
	if 12+17 != 29 {
		t.Fatal("unexpected total")
	}
}

func TestRows(t *testing.T) {
	t.Run("first", func(t *testing.T) {
		if len([]int{2, 4}) != 2 {
			t.Fatal("unexpected count")
		}
	})
}
