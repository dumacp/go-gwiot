package main

import (
	"encoding/json"

	"github.com/dumacp/go-gwiot/pkg/gwiotmsg"
)

func Discover(msg []byte) interface{} {

	subs := new(gwiotmsg.Discovery)

	if err := json.Unmarshal(msg, subs); err != nil {
		return err
	}

	return subs
}
