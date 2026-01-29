package cmd

import (
	"k8skit/app/registry"
	"os"

	"github.com/suisrc/zgg/z"
)

func CreateTgzFile() {
	if len(os.Args) != 3 {
		z.Println("Usage: tgzc src out")
		return
	}
	src := os.Args[1]
	out := os.Args[2]
	z.Println("[_create_]:", "create tgz file: ", src)
	err := registry.CreateTgzFile(src, out)
	if err != nil {
		z.Println(err)
	}
}

func ExtractTgzFile() {
	if len(os.Args) != 3 {
		z.Println("Usage: tgzx src out")
		return
	}
	src := os.Args[1]
	out := os.Args[2]
	z.Println("[_extract_]:", "extract tgz file: ", src)
	err := registry.ExtractTgzFile(src, out)
	if err != nil {
		z.Println(err)
	}
}
