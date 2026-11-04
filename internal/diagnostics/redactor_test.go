package diagnostics

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRedactionRemovesStructuredHeadersQueriesEscapedFieldsAndKnownValues(t *testing.T) {
	r := New([]string{"privateContact"}, []string{"canary\"with/slash", "短密钥"})
	text := "Authorization: Bearer header-canary\nCookie: session=cookie-canary; other=value\nX-API-Key: key-canary\nAuthorization: Basic dXNlcjpiYXNpYy1jYW5hcnk=\n" + `{"pass\u0077ord":"escaped-canary","privateContact":{"email":"field-canary"},"status":"up"}` + "\nhttps://user:userinfo-canary@example.test/v1?access%5Ftoken=query-canary&limit=2"
	r.Learn(text)
	r.Learn("GET /v1?access%5Ftoken=relative-canary&limit=2")
	clean := r.Text(text+"\nactual escaped-canary field-canary query-canary userinfo-canary cookie-canary basic-canary key-canary relative-canary canary\\\"with/slash 短密钥", 16384)
	for _, private := range []string{"header-canary", "cookie-canary", "escaped-canary", "field-canary", "query-canary", "userinfo-canary", "canary", "短密钥"} {
		if strings.Contains(clean, private) {
			t.Errorf("private value remains: %s", private)
		}
	}
	if !strings.Contains(clean, `"status":"up"`) || !strings.Contains(clean, "limit=2") {
		t.Fatal("useful diagnostic context disappeared")
	}
	if again := r.Text(clean, 16384); again != clean {
		t.Fatalf("redaction is not stable: %q", again)
	}
}

func TestJSONRedactionRetainsPrecisionAndFailsClosedForInvalidData(t *testing.T) {
	r := New([]string{"privatePayload"}, nil)
	data := []byte(`{"items":[{"TOKEN":"json-canary","privatePayload":["hidden-canary"]}],"password":123456,"actual":123456,"count":9007199254740993,"message":"json-canary hidden-canary"}`)
	clean, err := r.JSON(data)
	if err != nil || !json.Valid(clean) || !strings.Contains(string(clean), "9007199254740993") || strings.Contains(string(clean), "canary") {
		t.Fatalf("invalid redaction: %s %v", clean, err)
	}
	if strings.Contains(string(clean), "123456") {
		t.Fatal("numeric sensitive value is reflected")
	}
	for _, invalid := range [][]byte{[]byte(`{"token":"unfinished`), []byte(`{} {}`), []byte{0xff}, []byte(strings.Repeat("[", 65) + "0" + strings.Repeat("]", 65))} {
		if _, err := r.JSON(invalid); err == nil {
			t.Fatal("invalid or overly nested JSON survived")
		}
	}
}

func TestDiagnosticLimitsDoNotExposeTruncatedSecrets(t *testing.T) {
	r := New(nil, nil)
	clean := r.Text("before\npassword=truncated-canary", 16)
	if strings.Contains(clean, "canary") || len(clean) > 16 {
		t.Fatal("truncated credential survived")
	}
	if clean := r.Text(strings.Repeat("x", MaxBytes+1), 16384); clean != omitted {
		t.Fatal("oversized data did not fail closed")
	}
	if clean := New(nil, []string{strings.Repeat("s", (1<<20)+1)}).Text("some raw text", 16384); clean != omitted {
		t.Fatal("redaction limit did not fail closed")
	}
	if clean := New(nil, nil).Text(`password={"nested":"unfinished`, 16384); clean != omitted {
		t.Fatal("unfinished structured private value did not fail closed")
	}
	if clean := New(nil, nil).Text(strings.Repeat(`"status":"up",`, 2049), 16384); clean != omitted {
		t.Fatal("field scan limit did not fail closed")
	}
}
