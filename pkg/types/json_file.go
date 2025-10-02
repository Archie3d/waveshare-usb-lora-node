package types

import (
	"encoding/json"
	"io"
	"os"
)

// Read a JSON file and deserialize into a given type.
func LoadFromJsonFile[T any](path string, t *T) error {
	jsonFile, err := os.Open(path)
	if err != nil {
		return err
	}

	defer jsonFile.Close()

	bytes, err := io.ReadAll(jsonFile)
	if err != nil {
		return err
	}

	return json.Unmarshal(bytes, t)
}

// Serialize and save a JSON file.
func SaveToJsonFile[T any](path string, t *T) error {
	bytes, err := json.Marshal(t)
	if err != nil {
		return err
	}

	jsonFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer jsonFile.Close()

	_, err = jsonFile.Write(bytes)
	return err
}
