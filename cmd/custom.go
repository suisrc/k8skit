package cmd

import (
	"fmt"
	"kube-sidecar/z"
)

func init() {
	z.CmdR["hello"] = hello
}

func hello() {
	fmt.Println("hello world!")
}
