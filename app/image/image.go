package image

import (
	"flag"
	"k8skit/app/registry"
	"os"

	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/zc"
)

var (
	C = struct {
		Imagex registry.Config
	}{}
)

func init() {
	z.Config(&C)

	flag.BoolVar(&C.Imagex.Disable, "i5disable", false, "disable imagex")
	flag.StringVar(&C.Imagex.Username, "i5username", "", "imagex username")
	flag.StringVar(&C.Imagex.Password, "i5password", "", "imagex password")
	flag.StringVar(&C.Imagex.Image, "i5image", "", "imagex image name")
	flag.StringVar(&C.Imagex.SrcPath, "i5srcpath", "/opt/www", "imagex source path")
	flag.StringVar(&C.Imagex.OutPath, "i5outpath", "/opt/www", "imagex output path")

	z.Register("11-app.init", func(zgg *z.Zgg) z.Closed {
		if C.Imagex.Image == "" {
			z.Println("[_imagex_]: image name is empty, disable imagex")
		} else if C.Imagex.Disable {
			z.Println("[_imagex_]: imagex is disable", zc.CFG_ENV+"_IMAGEX_DISABLE=true")
		} else {
			z.Println("[_imagex_]: pull", C.Imagex.Image)
			// 创建输出目录
			if C.Imagex.OutPath != "" && C.Imagex.OutPath != "none" {
				if err := os.MkdirAll(C.Imagex.OutPath, 0666); err != nil {
					zgg.ServeStop("imagex, mkdir out dir:", err.Error())
					return nil
				}
			}
			// 提取镜像文件
			if err := registry.ExportFile(&C.Imagex); err != nil {
				zgg.ServeStop("imagex, extract image file:", err.Error())
				return nil
			}
			if C.Imagex.Version != "" {
				z.Version = C.Imagex.Version
			}
		}

		return nil
	})
}
