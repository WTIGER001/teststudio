package diagnostics

import (
	"testing"
	"time"
)

func TestAssertion(t *testing.T) { t.Fatal("intentional Go runner assertion diagnostic") }
func TestPanic(t *testing.T)     { panic("intentional Go runner panic diagnostic") }
func TestTimeout(t *testing.T)   { time.Sleep(30 * time.Second) }
func TestSkipped(t *testing.T)   { t.Skip("intentional Go runner skip diagnostic") }
