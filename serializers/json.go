package serializers

import (
	"encoding/json"
)

type JSONSerializer struct{}

func (s *JSONSerializer) Serialize(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (d *JSONSerializer) Deserialize(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
