package shared

import "testing"

func TestRenderJSON_MinifiedByDefault(t *testing.T) {
	out, err := RenderJSON(map[string]any{"a": 1}, false)
	if err != nil {
		t.Fatal(err)
	}

	if string(out) != "{\"a\":1}\n" {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestRenderJSON_Pretty(t *testing.T) {
	out, err := RenderJSON(map[string]any{"a": 1}, true)
	if err != nil {
		t.Fatal(err)
	}

	expected := "{\n  \"a\": 1\n}\n"
	if string(out) != expected {
		t.Fatalf("unexpected output: %q", out)
	}
}
