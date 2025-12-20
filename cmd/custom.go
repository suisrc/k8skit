package cmd

import (
	"fmt"

	"github.com/suisrc/zgg/z"
)

func init() {
	z.CMDR["world"] = world
}

func world() {
	fmt.Println("hello world!")
}
