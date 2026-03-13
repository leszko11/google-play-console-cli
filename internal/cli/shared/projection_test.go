package shared

import "testing"

func TestProjectFieldsNestedObject(t *testing.T) {
	got, err := ProjectFields(map[string]any{
		"packageName": "com.example.app",
		"product": map[string]any{
			"id": "coins_100",
			"listings": []any{
				map[string]any{"locale": "en-US", "title": "Coins"},
			},
		},
	}, "packageName,product.id,product.listings.0.locale")
	if err != nil {
		t.Fatal(err)
	}

	projected := got.(map[string]any)
	if projected["packageName"] != "com.example.app" {
		t.Fatalf("unexpected packageName: %#v", projected["packageName"])
	}
	product := projected["product"].(map[string]any)
	if product["id"] != "coins_100" {
		t.Fatalf("unexpected product id: %#v", product["id"])
	}
	listings := product["listings"].([]any)
	entry := listings[0].(map[string]any)
	if entry["locale"] != "en-US" {
		t.Fatalf("unexpected listing locale: %#v", entry["locale"])
	}
}

func TestProjectFieldsInvalidPath(t *testing.T) {
	_, err := ProjectFields(map[string]any{"a": 1}, "missing")
	if err == nil || err.Error() != `field path "missing" not found` {
		t.Fatalf("unexpected error: %v", err)
	}
}
