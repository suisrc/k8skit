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
	z.CMD["deploy"] = RunDeploy
}

func RunDeploy() {
	// parse command line arguments
	z.Initializ()
	var cfs string
	flag.StringVar(&cfs, "c", "", "config file path")
	flag.Parse()
	zc.LoadConfig(cfs)
	// upload to s3
	hfs := http.FS(os.DirFS("www"))
	err := s3cdn.UploadToS3(hfs, &front2.C.Front2, &s3cdn.C.S3cdn)
	if err != nil {
		z.Fatalln(err)
	} else {
		z.Println("Upload to S3 success")
	}
}
