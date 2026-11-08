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

func TestRecordedPageAddressCheckFollowsSelectedEnvironment(t *testing.T) {
	original := `await page.goto('https://recording.test/'); await expect(page).toHaveURL("https://recording.test/news?page=2#stories"); await expect(page).toHaveURL("https://elsewhere.test/news");`
	bound, err := bindRecordedNavigation(original, "https://recording.test/")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(bound, "process.env.TESTSTUDIO_BASE_URL") != 2 || !strings.Contains(bound, `toHaveURL(new URL("/news?page=2#stories", process.env.TESTSTUDIO_BASE_URL).href)`) || !strings.Contains(bound, `toHaveURL("https://elsewhere.test/news")`) {
		t.Fatal(bound)
	}
	if _, err := bindRecordedNavigation(`await expect(page).toHaveURL("https://recording.test/news")`, "https://recording.test/"); err == nil {
		t.Fatal("an address check replaced required navigation")
	}
}
