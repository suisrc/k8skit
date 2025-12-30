package cmd

import (
	"fmt"

	"github.com/suisrc/zgg/z"
)

// go run main.go world

func init() {
	z.CMDR["hello"] = hello
}

func hello() {
	fmt.Println("hello world!")
}
