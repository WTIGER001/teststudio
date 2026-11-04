package diagnostics

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const Version = "api-diagnostics/v1"
const Mask = "[REDACTED]"
const omitted = "[API diagnostic omitted: redaction limit]"
const MaxBytes = 2 << 20

var assignments = regexp.MustCompile(`("(?:\\.|[^"\\\r\n])*"|'[^'\r\n]*'|\b[A-Za-z][A-Za-z0-9_.-]{0,127})[ \t]*[:=][ \t]*`)
var addresses = regexp.MustCompile(`(?:https?://|/)[^\s"'<>]+`)
var escapes = regexp.MustCompile("\x1b(?:\\[[0-?]*[ -/]*[@-~]|\\][^\x07\x1b]*(?:\x07|\x1b\\\\))")

type Redactor struct {
	fields   map[string]bool
	values   map[string]bool
	bytes    int
	overflow bool
	replacer *strings.Replacer
}

func (r *Redactor) Limited() bool { return r.overflow }

func key(value string) string {
	return strings.ToLower(strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return -1
	}, value))
}

func New(fields, values []string) *Redactor {
	r := &Redactor{fields: map[string]bool{}, values: map[string]bool{}}
	for _, value := range strings.Fields("authorization proxy-authorization cookie set-cookie password passwd secret client-secret token access-token refresh-token id-token api-key x-api-key x-auth-token x-access-token csrf-token xsrf-token x-csrf-token x-xsrf-token x-amz-security-token x-goog-api-key credential credentials session session-id session-token") {
		r.fields[key(value)] = true
	}
	for _, value := range fields {
		r.fields[key(value)] = true
	}
	for _, value := range values {
		r.add(value)
	}
	return r
}

func (r *Redactor) add(value string) {
	if value == "" || value == Mask || r.values[value] {
		return
	}
	if len(r.values) >= 512 || r.bytes+len(value) > 1<<20 {
		r.overflow = true
		return
	}
	r.values[value] = true
	r.bytes += len(value)
	r.replacer = nil
}

func (r *Redactor) leaves(value any, depth int) {
	if depth > 64 {
		r.overflow = true
		return
	}
	switch value := value.(type) {
	case string:
		r.add(value)
	case json.Number:
		r.add(string(value))
	case []any:
		for _, item := range value {
			r.leaves(item, depth+1)
		}
	case map[string]any:
		for _, item := range value {
			r.leaves(item, depth+1)
		}
	}
}

func (r *Redactor) assignments(text string, learn bool) string {
	var output strings.Builder
	position := 0
	matches := assignments.FindAllStringSubmatchIndex(text, 2049)
	if len(matches) > 2048 {
		r.overflow = true
		return omitted
	}
	for _, match := range matches {
		if match[0] < position {
			continue
		}
		name := text[match[2]:match[3]]
		if strings.HasPrefix(name, "\"") {
			decoded, err := strconv.Unquote(name)
			if err != nil {
				continue
			}
			name = decoded
		} else {
			name = strings.Trim(name, "'")
		}
		if !r.fields[key(name)] {
			continue
		}
		start, end := match[1], match[1]
		var value any
		quoted := false
		if strings.HasPrefix(text[start:], Mask) {
			end = start + len(Mask)
			value = Mask
		} else if start < len(text) && strings.ContainsRune("\"{[", rune(text[start])) {
			decoder := json.NewDecoder(strings.NewReader(text[start:]))
			decoder.UseNumber()
			if decoder.Decode(&value) == nil {
				end = start + int(decoder.InputOffset())
				quoted = true
			} else {
				r.overflow = true
				return omitted
			}
		}
		if end == start {
			for end < len(text) && !strings.ContainsRune("\r\n,}&", rune(text[end])) {
				end++
			}
			value = strings.TrimSpace(strings.Trim(text[start:end], "'"))
		}
		if learn {
			r.leaves(value, 0)
			r.headerValues(key(name), value)
		}
		output.WriteString(text[position:start])
		if quoted {
			output.WriteString(strconv.Quote(Mask))
		} else {
			output.WriteString(Mask)
		}
		position = end
	}
	output.WriteString(text[position:])
	return output.String()
}

func (r *Redactor) headerValues(name string, value any) {
	if values, ok := value.([]any); ok {
		for _, value := range values {
			r.headerValues(name, value)
		}
		return
	}
	raw, ok := value.(string)
	if !ok {
		return
	}
	switch name {
	case "authorization", "proxyauthorization":
		parts := strings.Fields(raw)
		if len(parts) != 2 {
			return
		}
		r.add(parts[1])
		if strings.EqualFold(parts[0], "Basic") {
			if decoded, err := base64.StdEncoding.DecodeString(parts[1]); err == nil {
				r.add(string(decoded))
				if _, password, ok := strings.Cut(string(decoded), ":"); ok {
					r.add(password)
				}
			}
		}
	case "cookie", "setcookie":
		for _, pair := range strings.Split(raw, ";") {
			if _, value, ok := strings.Cut(pair, "="); ok {
				r.add(strings.TrimSpace(value))
			}
			if name == "setcookie" {
				break
			}
		}
	}
}

func (r *Redactor) urls(text string, learn bool) string {
	return addresses.ReplaceAllStringFunc(text, func(address string) string {
		parsed, err := url.Parse(address)
		if err != nil {
			return "[API URL omitted]"
		}
		if parsed.User != nil {
			if learn {
				r.add(parsed.User.Username())
				password, _ := parsed.User.Password()
				r.add(password)
			}
			parsed.User = url.User(Mask)
		}
		query, err := url.ParseQuery(parsed.RawQuery)
		if err != nil {
			parsed.RawQuery = "[query omitted]"
			return parsed.String()
		}
		changed := false
		for name, values := range query {
			if r.fields[key(name)] {
				if learn {
					for _, value := range values {
						r.add(value)
					}
				}
				query[name] = []string{Mask}
				changed = true
			}
		}
		if changed {
			parsed.RawQuery = query.Encode()
		}
		return parsed.String()
	})
}

func (r *Redactor) Learn(text string) {
	if len(text) > MaxBytes {
		r.overflow = true
		return
	}
	text = escapes.ReplaceAllString(text, "")
	r.urls(text, true)
	r.assignments(text, true)
}

func (r *Redactor) Text(text string, limit int) string {
	if len(text) > MaxBytes || !utf8.ValidString(text) {
		return omitted
	}
	r.Learn(text)
	if r.overflow {
		return omitted
	}
	text = escapes.ReplaceAllString(text, "")
	text = r.assignments(r.urls(text, false), false)
	if r.replacer == nil {
		values := make([]string, 0, len(r.values)*4)
		for value := range r.values {
			encoded, _ := json.Marshal(value)
			values = append(values, value, string(encoded[1:len(encoded)-1]), url.QueryEscape(value), url.PathEscape(value))
		}
		slices.SortFunc(values, func(a, b string) int { return len(b) - len(a) })
		pairs := []string{Mask, Mask}
		seen := map[string]bool{Mask: true}
		for _, value := range values {
			if value != "" && !seen[value] {
				pairs = append(pairs, value, Mask)
				seen[value] = true
			}
		}
		r.replacer = strings.NewReplacer(pairs...)
	}
	text = r.replacer.Replace(text)
	text = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, text)
	if limit < 0 {
		limit = 0
	}
	if len(text) > limit {
		end := limit
		for end > 0 && !utf8.ValidString(text[:end]) {
			end--
		}
		text = text[:end]
	}
	return text
}

func (r *Redactor) JSON(data []byte) ([]byte, error) {
	if len(data) > MaxBytes || !utf8.Valid(data) {
		return nil, errors.New("API JSON diagnostic exceeds its redaction limit")
	}
	var value any
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	if decoder.Decode(&value) != nil {
		return nil, errors.New("API JSON diagnostic is invalid")
	}
	var extra any
	if !errors.Is(decoder.Decode(&extra), io.EOF) {
		return nil, errors.New("API JSON diagnostic has trailing data")
	}
	r.Learn(string(data))
	if r.overflow {
		return nil, errors.New("API JSON diagnostic exceeds its redaction limit")
	}
	var walk func(any, int) (any, error)
	walk = func(value any, depth int) (any, error) {
		if depth > 64 {
			return nil, errors.New("API JSON diagnostic exceeds its nesting limit")
		}
		switch value := value.(type) {
		case map[string]any:
			for name, item := range value {
				if r.fields[key(name)] {
					value[name] = Mask
					continue
				}
				clean, err := walk(item, depth+1)
				if err != nil {
					return nil, err
				}
				value[name] = clean
			}
		case []any:
			for index, item := range value {
				clean, err := walk(item, depth+1)
				if err != nil {
					return nil, err
				}
				value[index] = clean
			}
		case string:
			return r.Text(value, MaxBytes), nil
		case json.Number:
			if r.values[string(value)] {
				return Mask, nil
			}
		}
		return value, nil
	}
	value, err := walk(value, 0)
	if err != nil {
		return nil, err
	}
	return json.Marshal(value)
}
