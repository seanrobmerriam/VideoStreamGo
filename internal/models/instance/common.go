package instance

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// JSONMap is a custom type for JSONB columns
type JSONMap map[string]interface{}

func (j *JSONMap) Scan(val interface{}) error {
	var bytes []byte
	switch v := val.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	case nil:
		*j = JSONMap{}
		return nil
	default:
		return fmt.Errorf("unsupported type: %T", val)
	}
	if len(bytes) == 0 {
		*j = JSONMap{}
		return nil
	}
	return json.Unmarshal(bytes, j)
}

func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return "{}", nil
	}
	b, err := json.Marshal(j)
	return string(b), err
}

// JSONStringArray is a custom type for JSONB string array columns
type JSONStringArray []string

func (j *JSONStringArray) Scan(val interface{}) error {
	var bytes []byte
	switch v := val.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	case nil:
		*j = JSONStringArray{}
		return nil
	default:
		return fmt.Errorf("unsupported type: %T", val)
	}
	if len(bytes) == 0 {
		*j = JSONStringArray{}
		return nil
	}
	return json.Unmarshal(bytes, j)
}

func (j JSONStringArray) Value() (driver.Value, error) {
	if j == nil {
		return "[]", nil
	}
	b, err := json.Marshal(j)
	return string(b), err
}
