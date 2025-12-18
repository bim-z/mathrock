package auth

import (
	"encoding/json"
	"os/user"

	"github.com/zalando/go-keyring"
)

func Bearer() (token string, err error) {
	curr := new(current)

	usr, err := user.Current()
	if err != nil {
		return
	}

	tkn, err := keyring.Get("starducc", usr.Name)
	if err != nil {
		return
	}

	if err = json.Unmarshal([]byte(tkn), curr); err != nil {
		return
	}

	return curr.Token, nil
}
