package lifecycle

import (
	"testing"
	"time"
)

func TestCancellationWindow(t *testing.T) {
	t.Log("The cancellation window is open.")
	time.Sleep(40 * time.Second)
}

func TestTimeoutWindow(t *testing.T) {
	t.Log("This bounded check intentionally exceeds its catalog timeout.")
	time.Sleep(15 * time.Second)
}

func TestRecoveryWindow(t *testing.T) {
	t.Log("The controller interruption window is open.")
	time.Sleep(20 * time.Second)
}
