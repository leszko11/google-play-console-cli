package shared

import "testing"

func BenchmarkProjectFields(b *testing.B) {
	payload := map[string]any{
		"packageName": "com.example.app",
		"releases": []map[string]any{
			{
				"name":         "1.2.3",
				"versionCodes": []int{123, 124},
				"notes": []map[string]any{
					{"language": "en-US", "text": "Hello"},
					{"language": "fr-FR", "text": "Bonjour"},
				},
			},
		},
		"counts": map[string]any{
			"products":      25,
			"subscriptions": 10,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := ProjectFields(payload, "packageName,releases.0.versionCodes,counts.products"); err != nil {
			b.Fatalf("project fields: %v", err)
		}
	}
}
