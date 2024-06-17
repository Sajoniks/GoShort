package helper

import (
	"bytes"
	"encoding/json"
	"net/http"
)

func WriteJson(w http.ResponseWriter, v any) error {
	b := bytes.Buffer{}
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(true)
	enc.SetIndent("", "")
	if err := enc.Encode(v); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	_, err := w.Write(b.Bytes())
	return err
}

func WriteProblemJson(w http.ResponseWriter, v any) error {
	b := bytes.Buffer{}
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(true)
	enc.SetIndent("", "")
	if err := enc.Encode(v); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/problem+json")
	_, err := w.Write(b.Bytes())
	return err
}
