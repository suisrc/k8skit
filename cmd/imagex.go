package cmd

import (
	"flag"
	"k8skit/app/image"
	"k8skit/app/registry"
	"os"

	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/zc"
)

// 通过 imagex 命令，导出镜像文件

func ExpImageX() {
	var cfs string
	z.Initializ()
	flag.StringVar(&cfs, "c", "", "config file path")
	flag.Parse()
	zc.LoadConfig(cfs)
	if image.C.Imagex.OutPath != "" && image.C.Imagex.OutPath != "none" {
		if err := os.MkdirAll(image.C.Imagex.OutPath, 0666); err != nil {
			z.Exit("mkdir out dir:", err)
		}
	}
	if err := registry.ExportImage(&image.C.Imagex); err != nil {
		z.Exit("extract image file:", err)
	}
	z.Logn(z.ToStr2(image.C))
}
