package s3cdn

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/suisrc/zgg/app/front2"
	"github.com/suisrc/zgg/z"
)

// 上传到 S3
func UploadToS3(hfs http.FileSystem, fim map[string]fs.FileInfo, ffc *front2.Config, cfg *Config, app, ver string) error {
	ctx := context.Background()
	cli, err := GetClient(ctx, cfg)
	if err != nil {
		return err
	}
	rootdir := filepath.Join(cfg.RootDir, app, ver)
	cnamepath := filepath.Join(rootdir, "cname")
	cnametext := cfg.Domain + "/" + rootdir
	if strings.HasPrefix(cnametext, "https:") {
		cnametext = cnametext[6:]
	} else if strings.HasPrefix(cnametext, "http:") {
		cnametext = cnametext[5:]
	} else if !strings.HasPrefix(cnametext, "//") {
		cnametext = "//" + cnametext
	}
	if !cfg.Rewrite {
		// 判断 index 文件是否存在，如果存在，跳过上传
		// name := filepath.Join(rootdir, ffc.Index)
		// _, err = cli.StatObject(ctx, cfg.Bucket, name, minio.StatObjectOptions{})
		// if err == nil {
		// 	z.Println("[_cdnskip]: upload to s3:", name, ", exists")
		// 	return nil
		// }
		// if minio.ToErrorResponse(err).Code != "NoSuchKey" {
		// 	return err
		// }
		// z.Println("[_cdnskip]: upload to s3:", err.Error(), ", noskip")
		// obj := <-cli.ListObjects(ctx, cfg.Bucket, minio.ListObjectsOptions{MaxKeys: 1, Prefix: rootdir})
		obj, err := cli.GetObject(ctx, cfg.Bucket, cnamepath, minio.GetObjectOptions{})
		if err == nil {
			bts, err := io.ReadAll(obj)
			if err != nil {
				if strings.ToLower(minio.ToErrorResponse(err).Code) != "nosuchkey" {
					z.Println("[_cdnskip]: upload to s3:", cnamepath, ", read error:", err.Error())
					return err
				}
			} else {
				if string(bts) == cnametext {
					z.Println("[_cdnskip]: upload to s3:", cnamepath, ", exists")
					return nil // skip
				} else {
					z.Println("[_cdnskip]: upload to s3: cname no same", cnamepath, cnametext, string(bts))
				}
			}
		} else if strings.ToLower(minio.ToErrorResponse(err).Code) != "nosuchkey" {
			z.Println("[_cdnskip]: upload to s3:", cnamepath, ", get object error:", err.Error())
			return err
		}
		z.Println("[_cdnskip]: upload to s3:", cnamepath, ", noskip")
	}
	cnamebyte := bytes.NewReader([]byte(cnametext))
	cnamesize := cnamebyte.Size()
	if _, err = cli.PutObject(ctx, cfg.Bucket, cnamepath, cnamebyte, cnamesize, minio.PutObjectOptions{ContentType: "text/plain"}); err != nil {
		return err // panic(err) 上传标记域名文件
	}
	z.Println("[_success]: upload to s3:", cnamepath)
	// return UploadToS3Loop(ctx, cli, cfg.Bucket, rootdir, cfg.Domain, hfs, ffc)
	// 遍历所有的文件夹执行上传 hfs.Readdir(-1)
	for fpath, fstat := range fim {
		if fpath == "cname" {
			continue
		}
		if err := _UploadToS3(ctx, cli, fpath, fstat, rootdir, cnametext, hfs, fim, ffc, cfg); err != nil {
			return err
		}
	}
	return nil
}

func _UploadToS3(ctx context.Context, cli *minio.Client, fpath string, fstat fs.FileInfo, rootdir, rootpath string, //
	hfs http.FileSystem, fim map[string]fs.FileInfo, ffc *front2.Config, cfg *Config) error {
	file, err := hfs.Open("/" + fpath)
	if err != nil {
		z.Println("[_cdn_put_]: upload to s3:", fpath, ", open error:", err.Error())
		return err
	}
	defer file.Close()
	var rbts io.Reader
	var size int64
	isrp := false
	if front2.IsFixFile(fstat.Name(), ffc) {
		tbts, err := front2.GetFixFile(file, fstat.Name(), ffc.TmplRoot, rootpath, fim)
		if err != nil {
			z.Println("[_cdn_put_]: upload to s3:", fpath, ", read error:", err.Error())
			return err
		}
		rbts = bytes.NewReader(tbts)
		size = int64(len(tbts))
		isrp = true
	} else {
		rbts = file
		size = fstat.Size()
	}
	objname := filepath.Join(rootdir, fpath)
	objtype := mime.TypeByExtension(filepath.Ext(fstat.Name())) // 获取文件类型
	// z.Println("[========] upload to s3:", objname, "|", objtype)
	_, err = cli.PutObject(ctx, cfg.Bucket, objname, rbts, size, minio.PutObjectOptions{ContentType: objtype})
	if err != nil {
		z.Println("[_cdn_put_]: upload to s3:", fpath, ", put error:", err.Error())
		return err
	}
	if isrp {
		z.Println("[_replace]: upload to s3:", objname)
	} else {
		z.Println("[_success]: upload to s3:", objname)
	}
	return nil
}
