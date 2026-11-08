package sources

import (
	"encoding/json"
	"errors"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/wtiger001/teststudio/internal/browserpolicy"
)

var recordedNavigation = regexp.MustCompile(`(?:page\.goto\(|expect\(page\)\.toHaveURL\()('(?:[^'\\\r\n]|\\.)*'|"(?:[^"\\\r\n]|\\.)*")`)

func bindRecordedNavigation(code, address string) (string, error) {
	start, err := url.Parse(address)
	if err != nil {
		return "", err
	}
	origin, err := browserpolicy.Origin(start.Scheme + "://" + start.Host)
	if err != nil {
		return "", err
	}
	matches := recordedNavigation.FindAllStringSubmatchIndex(code, -1)
	bound := 0
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]
		value, err := recordedURLLiteral(code[match[2]:match[3]])
		if err != nil {
			return "", err
		}
		u, err := url.Parse(value)
		if err != nil {
			return "", err
		}
		actual, err := browserpolicy.Origin(u.Scheme + "://" + u.Host)
		if err != nil || actual != origin {
			continue
		}
		u.Scheme, u.Host = "", ""
		path := u.String()
		if path == "" {
			path = "/"
		}
		encoded, _ := json.Marshal(path)
		replacement := "new URL(" + string(encoded) + ", process.env.TESTSTUDIO_BASE_URL).href"
		if strings.HasPrefix(code[match[0]:match[1]], "page.goto(") {
			bound++
		}
		code = code[:match[2]] + replacement + code[match[3]:]
	}
	if bound == 0 {
		return "", errors.New("the recording has no supported start-origin navigation; keep the retained code and add explicit target binding in the source editor")
	}
	return code, nil
}

func recordedURLLiteral(literal string) (string, error) {
	if strings.HasPrefix(literal, `"`) {
		return strconv.Unquote(literal)
	}
	var value strings.Builder
	for i := 1; i < len(literal)-1; i++ {
		c := literal[i]
		if c == '\\' {
			i++
			if i >= len(literal)-1 || !strings.ContainsRune(`\'"/`, rune(literal[i])) {
				return "", errors.New("recorded navigation requires a simple URL literal; review the retained original source")
			}
			c = literal[i]
		}
		value.WriteByte(c)
	}
	return value.String(), nil
}
