package recording

import "testing"

func TestInteractiveActionsBoundChecksAndViewport(t *testing.T) {
	valid := []Action{
		{Action: "resize", Width: 320, Height: 240}, {Action: "resize", Width: 1920, Height: 1200},
		{Action: "click", X: 1800, Y: 1100}, {Action: "assert-pick"}, {Action: "assert-url"},
		{Action: "assert-add", Key: "at-least", Count: 1}, {Action: "assert-add", Key: "exactly", Count: 0},
		{Action: "assert-add", Key: "text", Text: "Regional harvest"},
		{Action: "assert-add", Key: "at-most", Count: 2, Scope: "group"},
		{Action: "assert-add", Key: "exactly", Count: 1, Scope: "element"},
		{Action: "assert-add", Key: "exact-text", Text: "Regional harvest"},
		{Action: "assert-add", Key: "value", Text: ""},
		{Action: "assert-add", Key: "visible"}, {Action: "assert-add", Key: "enabled"},
		{Action: "assert-add", Key: "disabled"}, {Action: "assert-add", Key: "checked"},
		{Action: "assert-add", Key: "unchecked"}, {Action: "assert-title"},
	}
	invalid := []Action{
		{Action: "assert-add", Key: "visible", Text: "extra"},
		{Action: "assert-add", Key: "enabled", Count: 1},
		{Action: "assert-add", Key: "value", Scope: "group"},
		{Action: "assert-add", Key: "at-most", Count: 1, Scope: "arbitrary"},
		{Action: "click", Scope: "element"},
		{Action: "resize", Width: 319, Height: 720}, {Action: "resize", Width: 1921, Height: 720},
		{Action: "resize", Width: 1280, Height: 1201}, {Action: "resize", Width: 1280, Height: 239},
		{Action: "click", X: 1920, Y: 0}, {Action: "click", Y: 1200},
		{Action: "assert-add", Key: "at-least", Count: 0}, {Action: "assert-add", Key: "exactly", Count: -1},
		{Action: "assert-add", Key: "exactly", Count: 10001}, {Action: "assert-add", Key: "text"},
		{Action: "assert-add", Key: "text", Text: "a", Count: 1}, {Action: "assert-url", Width: 1280},
		{Action: "resize", Width: 1280, Height: 720, Text: "unexpected"}, {Action: "assert-pick", Key: "unexpected"},
	}
	for _, action := range valid {
		action.Sequence = 1
		if !action.Valid() {
			t.Errorf("valid action rejected: %+v", action)
		}
	}
	for _, action := range invalid {
		action.Sequence = 1
		if action.Valid() {
			t.Errorf("invalid action accepted: %+v", action)
		}
	}
}
