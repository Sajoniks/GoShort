package helper

import (
	"encoding/json"
	"io"
)

func DecodeJson(r io.Reader, v any) error {
	defer io.Copy(io.Discard, r)
	return json.NewDecoder(r).Decode(v)
}
