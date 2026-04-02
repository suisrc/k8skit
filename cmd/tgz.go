package cmd

import (
	"k8skit/app/registry"
	"os"
	"strings"

	"github.com/suisrc/zgg/z"
)

func CreateTgzFile() {
	if len(os.Args) != 3 {
		z.Logn("Usage: tgzc src out")
		return
	}
	src := os.Args[1]
	out := os.Args[2]
	z.Logn("[_create_]:", "create tgz file: ", src)
	err := registry.CreateTgzFile(src, out)
	if err != nil {
		z.Logn(err)
	}
}

func ExtractTgzFile() {
	if len(os.Args) != 3 {
		z.Logn("Usage: tgzx src out")
		return
	}
	src := os.Args[1]
	out := os.Args[2]
	z.Logn("[_extract]:", "extract tgz file: ", src)
	err := registry.ExtractTgzFile(src, "", out)
	if err != nil {
		z.Logn(err)
	}
}

func ExtractTgzResp() {
	if len(os.Args) < 3 {
		z.Logn("Usage: tgzx src out tar")
		return
	}
	src := os.Args[1]
	out := os.Args[2]
	var tar bool
	if len(os.Args) > 3 && os.Args[3] == "tar" {
		tar = true
	}
	if tar && strings.HasPrefix(src, "git+") {
		z.Logn("[_extract]:", "create tgz file: ", out+".tgz")
		err := registry.CreateTgzFileByGit(out+".tgz", "", "", src[4:])
		if err != nil {
			z.Logn("[_extract]: download by git error:", err.Error())
			return
		}
	} else {
		z.Logn("[_extract]:", "extract tgz file: ", out)
		err := registry.GetFilesByGitOrTgz(out, "", src)
		if err != nil {
			z.Logn("[_extract]: download by http error:", err.Error())
			return
		}
	}
	z.Logn("[_extract]: success")
}
