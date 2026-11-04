package sources

import (
	"strings"
	"testing"
)

func TestRecordedNavigationRebindingKeepsForeignOriginsVisible(t *testing.T) {
	original := `await page.goto('http://localhost:3000/cart?item=1'); await page.goto("http://localhost:3000/checkout"); await page.goto('https://external.test/');`
	bound, err := bindRecordedNavigation(original, "http://localhost:3000/")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(bound, "process.env.TESTSTUDIO_BASE_URL") != 2 || !strings.Contains(bound, `"/cart?item=1"`) || !strings.Contains(bound, `page.goto('https://external.test/')`) {
		t.Fatal(bound)
	}
	if _, err := bindRecordedNavigation(`await page.setContent('<h1>Offline</h1>')`, "http://localhost:3000/"); err == nil {
		t.Fatal("non-recorded source was silently rebound")
	}
}
