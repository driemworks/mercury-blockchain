package core

import (
	"bytes"
	"encoding/json"
)

/**
Pretty print json with indentation
*/
func PrettyPrintJSON(messageJSONBytes []byte) (bytes.Buffer, error) {
	var prettyJSON bytes.Buffer
	err := json.Indent(&prettyJSON, messageJSONBytes, "", "\t")
	if err != nil {
		return bytes.Buffer{}, err
	}
	return prettyJSON, nil
}
