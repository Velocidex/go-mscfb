package parser

import (
	"encoding/json"

	"github.com/davecgh/go-spew/spew"
)

func Debug(arg interface{}) {
	spew.Dump(arg)
}

func Dump(in interface{}) string {
	serialized, _ := json.Marshal(in)
	return string(serialized) + "\n"
}
