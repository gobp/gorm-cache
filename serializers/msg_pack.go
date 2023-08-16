package serializers

import (
	"github.com/vmihailenco/msgpack/v5"
)

type MsgPackSerializer struct{}

func (s *MsgPackSerializer) Serialize(v any) ([]byte, error) {
	return msgpack.Marshal(v)
}

func (d *MsgPackSerializer) Deserialize(data []byte, v any) error {
	return msgpack.Unmarshal(data, v)
}
