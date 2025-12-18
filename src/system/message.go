package system

import (
	"encoding/json"
	"io"
)

type message struct {
	Message string `json:"message"`
}

func Parse(body io.ReadCloser) (msg string, err error) {
	mess := new(message)

	data := []byte{}

	if _, err = body.Read(data); err != nil {
		return
	}

	if err = json.Unmarshal(data, mess); err != nil {
		return
	}

	return mess.Message, nil
}
