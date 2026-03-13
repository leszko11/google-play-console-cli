package shared

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

func ProjectFields(v any, rawFields string) (any, error) {
	fields := strings.Split(strings.TrimSpace(rawFields), ",")
	normalized := make([][]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		normalized = append(normalized, strings.Split(field, "."))
	}
	if len(normalized) == 0 {
		return nil, fmt.Errorf("--fields must contain at least one path")
	}

	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	var doc any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}

	projected, err := projectNode(doc, normalized)
	if err != nil {
		return nil, err
	}
	return projected, nil
}

func projectNode(node any, paths [][]string) (any, error) {
	switch typed := node.(type) {
	case map[string]any:
		out := make(map[string]any)
		for _, path := range paths {
			if len(path) == 0 {
				continue
			}
			key := path[0]
			child, ok := typed[key]
			if !ok {
				return nil, fmt.Errorf("field path %q not found", strings.Join(path, "."))
			}
			if len(path) == 1 {
				out[key] = child
				continue
			}
			value, err := projectNode(child, [][]string{path[1:]})
			if err != nil {
				return nil, err
			}
			if existing, ok := out[key]; ok {
				merged, err := mergeProjected(existing, value)
				if err != nil {
					return nil, err
				}
				out[key] = merged
			} else {
				out[key] = value
			}
		}
		return out, nil
	case []any:
		out := make([]any, len(typed))
		for _, path := range paths {
			if len(path) == 0 {
				continue
			}
			index, err := strconv.Atoi(path[0])
			if err != nil {
				return nil, fmt.Errorf("field path %q uses non-numeric array index", strings.Join(path, "."))
			}
			if index < 0 || index >= len(typed) {
				return nil, fmt.Errorf("field path %q index out of range", strings.Join(path, "."))
			}
			if len(path) == 1 {
				out[index] = typed[index]
				continue
			}
			value, err := projectNode(typed[index], [][]string{path[1:]})
			if err != nil {
				return nil, err
			}
			if out[index] != nil {
				merged, err := mergeProjected(out[index], value)
				if err != nil {
					return nil, err
				}
				out[index] = merged
			} else {
				out[index] = value
			}
		}
		return out, nil
	default:
		return nil, fmt.Errorf("field path descends into scalar value")
	}
}

func mergeProjected(existing, incoming any) (any, error) {
	switch left := existing.(type) {
	case map[string]any:
		right, ok := incoming.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("cannot merge projected field types")
		}
		for k, v := range right {
			if prev, ok := left[k]; ok {
				merged, err := mergeProjected(prev, v)
				if err != nil {
					return nil, err
				}
				left[k] = merged
				continue
			}
			left[k] = v
		}
		return left, nil
	case []any:
		right, ok := incoming.([]any)
		if !ok || len(right) != len(left) {
			return nil, fmt.Errorf("cannot merge projected field types")
		}
		for i := range right {
			if right[i] == nil {
				continue
			}
			if left[i] == nil {
				left[i] = right[i]
				continue
			}
			merged, err := mergeProjected(left[i], right[i])
			if err != nil {
				return nil, err
			}
			left[i] = merged
		}
		return left, nil
	default:
		return incoming, nil
	}
}
