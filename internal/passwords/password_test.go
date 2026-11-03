package passwords

import (
	"context"
	"strings"
	"testing"
)

func TestPasswordHashingUnicodeAndMalformedCosts(t *testing.T) {
	password := "café silver sunrise 42"
	encoded, err := Hash(t.Context(), password)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Hash(t.Context(), password)
	if err != nil || encoded == second {
		t.Fatal("password salt was reused", err)
	}
	if strings.Contains(encoded, password) || !ValidHash(encoded) {
		t.Fatal("invalid password encoding")
	}
	for _, value := range []string{password, "cafe\u0301 silver sunrise 42"} {
		valid, err := Verify(t.Context(), encoded, value)
		if err != nil || !valid {
			t.Fatal("equivalent normalized password failed", err)
		}
	}
	valid, err := Verify(t.Context(), encoded, "different silver sunrise 42")
	if err != nil || valid {
		t.Fatal("incorrect password accepted", err)
	}
	for _, bad := range []string{"", strings.Replace(encoded, "m=65536", "m=4294967295", 1), strings.Replace(encoded, "v=19", "v=18", 1), encoded + "$extra"} {
		if ValidHash(bad) {
			t.Fatal("invalid encoded cost accepted")
		}
	}
	for _, bad := range []string{"short", strings.Repeat("a", 15), "passwordpassword", "line\nbreak with numbers", strings.Repeat("漢", 257), strings.Repeat("\u1100\u1161", 256)} {
		if err := Validate(bad); err == nil {
			t.Fatal("invalid password accepted")
		}
	}
	if err := Validate(" a long phrase with spaces "); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := Hash(ctx, "cancelled sample secret 42"); err == nil {
		t.Fatal("cancelled hash succeeded")
	}
}
