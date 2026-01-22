package cmd

import (
	"flag"
	"k8skit/app"
	"k8skit/app/registry"
	"os"

	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/zc"
)

func RunImageX() {
	var cfs string
	z.Initializ()
	flag.StringVar(&cfs, "c", "", "config file path")
	flag.Parse()
	zc.LoadConfig(cfs)
	if err := os.MkdirAll(app.C.Imagex.OutPath, 0666); err != nil {
		z.Fatalf("mkdir out dir: %v", err)
	}
	if err := registry.ExtractImageFile(&app.C.Imagex); err != nil {
		z.Fatalf("extract image file: %v", err)
	}
	z.Println(z.ToStr2(app.C))
}
