package cmd

import (
	"flag"
	"k8skit/app/s3cdn"
	"net/http"
	"os"

	"github.com/suisrc/zgg/app/front2"
	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/zc"
)

func init() {
	z.CMD["deploy"] = ExeDeploy
	z.CMD["imagex"] = ExpImageX
	z.CMD["tgzc"] = CreateTgzFile  // 压缩文件
	z.CMD["tgzx"] = ExtractTgzFile // 解压文件
}

func ExeDeploy() {
	// parse command line arguments
	z.Initializ()
	var cfs string
	var www string
	flag.StringVar(&cfs, "c", "", "config file path")
	flag.StringVar(&www, "www", "www", "www directory")
	flag.Parse()
	zc.LoadConfig(cfs)
	// upload to s3
	ffs := os.DirFS(www)
	fim, err := front2.GetFileMap(ffs)
	if err != nil {
		z.Fatalln(err)
	}
	hfs := http.FS(ffs)
	err = s3cdn.UploadToS3(hfs, fim, &front2.C.Front2, &s3cdn.C.S3cdn, z.AppName, z.Version)
	if err != nil {
		z.Fatalln(err)
	}
	z.Println("[_deploy_]:", "upload to S3 success")
}
