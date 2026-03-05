package cmd

import (
	"fmt"

	"github.com/suisrc/zgg/z"
)

// go run main.go world

func init() {
	z.CMD["sync"] = sync
}

func sync() {
	fmt.Println("sync k8s ...")

}
