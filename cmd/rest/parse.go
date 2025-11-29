package rest

import (
	"encoding/json"

	"resty.dev/v3"
)

func Parse(res *resty.Response) (r *Response, err error) {
	r = new(Response)

	body := []byte{}

	if _, err = res.Body.Read(body); err != nil {
		return
	}

	if err = json.Unmarshal(body, res); err != nil {
		return
	}

	return
}
