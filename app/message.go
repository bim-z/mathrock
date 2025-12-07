package main

import (
	"encoding/json"
	"io"
)

type message struct {
	Msg string `json:"message"`
}

func parse(body io.ReadCloser) (msg string, err error) {
	mess := new(message)

	data := []byte{}

	if _, err = body.Read(data); err != nil {
		return
	}

	if err = json.Unmarshal(data, mess); err != nil {
		return
	}

	return mess.Msg, nil
}
