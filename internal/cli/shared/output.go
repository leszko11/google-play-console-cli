package shared

import "encoding/json"

func RenderJSON(v any, pretty bool) ([]byte, error) {
	if pretty {
		out, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return nil, err
		}
		return append(out, '\n'), nil
	}

	out, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	return append(out, '\n'), nil
}
