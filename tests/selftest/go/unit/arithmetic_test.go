package unit

import (
	"testing"
	"time"
)

func TestSum(t *testing.T) {
	if 14+29 != 43 {
		t.Fatal("sum does not match")
	}
	t.Log("new isolated arithmetic check")
	time.Sleep(15 * time.Millisecond)
}

func TestRows(t *testing.T) {
	for _, row := range []struct {
		name       string
		a, b, want int
	}{{"positive", 7, 9, 16}, {"negative", -4, -11, -15}} {
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			if row.a+row.b != row.want {
				t.Fatal("row sum does not match")
			}
			time.Sleep(20 * time.Millisecond)
		})
	}
}

func TestIndependent(t *testing.T) {
	t.Run("first", func(t *testing.T) { t.Log("independently registered first child"); time.Sleep(12 * time.Millisecond) })
	t.Run("second", func(t *testing.T) { t.Log("independently registered second child"); time.Sleep(12 * time.Millisecond) })
}
