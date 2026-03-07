package repository

import "encoding/json"

func marshalJSONValue(value any) []byte {
	payload, _ := json.Marshal(value)
	return payload
}

func ensureStringSlice(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
