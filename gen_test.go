package main

import (
	"encoding/json"
	"io/fs"
	"io/ioutil"
	"testing"
)

func TestGen(t *testing.T) {
	o := []opsModel{
		{
			NameSpace: "ak-dev",
			Servers: []serverConf{
				{
					Name:       "user-info",
					ServerPath: "/root/mobile/ak/user-info",
				},
				{
					Name:       "jzj-7201",
					ServerPath: "/root/mobile/ak/jzj-7201",
				},
			},
		},
	}
	raw, err := json.Marshal(o)
	if err!=nil{
		return
	}
	t.Log(ioutil.WriteFile("ops.json", raw, fs.ModePerm))
}
