package logger

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/spf13/cast"
	"github.com/stretchr/testify/assert"
)

func TestAccessLog(t *testing.T) {

	url := "example.com"
	content := `{"integer":1,"string":"xyz", "object": { "element": 1 } , "array": [1, 2, 3]}`
	byte := []byte(content)

	body := bytes.NewReader(byte)
	req, _ := http.NewRequest("POST", url, body)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Length", cast.ToString(len(content)))

	result := dumpJsonBoddy(req)
	assert.Equal(t, result, "map[array:[1 2 3] integer:1 object:map[element:1] string:xyz]")
}
