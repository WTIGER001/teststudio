package gosuite

import (
	"regexp"
	"strings"
	"testing"
)

func TestSelectorsAreExplicitAndCannotInjectGoFlags(t *testing.T) {
	for _, selector := range []string{".::Test", "./unit::TestSum", "./unit::TestCases/with_+_symbols", ".::Example"} {
		if _, err := Parse(selector); err != nil {
			t.Fatalf("valid selector %q: %v", selector, err)
		}
	}
	for _, selector := range []string{"./...::TestAll", "../other::TestAll", "./a/../b::TestAll", "./a/...::TestAll", "-exec=sh::TestAll", "github.com/other::TestAll", ".::TestAll|TestOther", ".::TestAll;cmd", ".::TestAll/.*", ".::TestAll\n", strings.Repeat("a", 1025)} {
		if _, err := Parse(selector); err == nil {
			t.Fatalf("unsafe selector accepted: %q", selector)
		}
	}
	parts := strings.Split(Pattern("TestRows/plus+value/eq=a.b"), "/")
	for i, want := range []string{"TestRows", "plus+value", "eq=a.b"} {
		r := regexp.MustCompile(parts[i])
		if !r.MatchString(want) || r.MatchString("prefix"+want) || r.MatchString(want+"suffix") {
			t.Fatalf("selector is not exact: %s", parts[i])
		}
	}
}

func TestManifestRegistersParentsOrIndependentChildrenWithoutInflatedCounts(t *testing.T) {
	parent := Entry{ID: "GO-1", Package: "./unit", Test: "TestRows"}
	a := Entry{ID: "GO-2", Package: "./unit", Test: "TestRows/a"}
	b := Entry{ID: "GO-3", Package: "./unit", Test: "TestRows/b"}
	if err := Validate([]Entry{a, b}); err != nil {
		t.Fatal(err)
	}
	for _, entries := range [][]Entry{nil, {parent, a}, {a, parent}, {a, a}, {parent, {ID: "GO-2", Package: "./unit", Test: "TestRows"}}} {
		if err := Validate(entries); err == nil {
			t.Fatalf("ambiguous registration accepted: %+v", entries)
		}
	}
	b.Package = "./other"
	b.Test = parent.Test
	if err := Validate([]Entry{parent, b}); err != nil {
		t.Fatal(err)
	}
}
