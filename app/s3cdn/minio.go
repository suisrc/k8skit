package s3cdn

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/suisrc/zgg/app/front2"
	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/zc"
)

var (
	C = struct {
		S3cdn S3cdnConfig
	}{}
)

type S3cdnConfig struct {
	Enable   bool   `json:"enable"`   // 禁用
	Access   string `json:"access"`   // 账号
	Secret   string `json:"secret"`   // 秘钥
	TToken   string `json:"token"`    // 临时令牌
	Signer   int    `json:"signer"`   // 签名, 0: default, 1: v4, 2: v2, 3: v4stream, 4: anonymous
	Endpoint string `json:"endpoint"` // 接口
	Region   string `json:"region"`   // 区域
	Bucket   string `json:"bucket"`   // 存储桶
	RootDir  string `json:"rootdir"`  // 根目录
	Domain   string `json:"domain"`   // CDN域名， 包含桶信息，比如 //[bucket].x.y.z or //x.y.z/[bucket]
	Rewrite  bool   `json:"rewrite"`  // 是否覆盖, 重写前端信息到CDN上，不会删除，但是会覆盖相同的名称
	AddrPort string `json:"addrport"` // 监听端口，不破坏源服务，独立新服务监控 CDN 索引
}

func init() {
	z.Config(&C)

	flag.BoolVar(&C.S3cdn.Enable, "s3enable", false, "S3 启用")
	flag.StringVar(&C.S3cdn.Access, "s3access", "", "S3 账号")
	flag.StringVar(&C.S3cdn.Secret, "s3secret", "", "S3 秘钥")
	flag.Var(z.NewStrVal(&C.S3cdn.TToken, ""), "s3ttoken", "S3 临时令牌")
	flag.IntVar(&C.S3cdn.Signer, "s3signer", 1, "S3 签名, 0: def, 1: v4(default), 2: v2, 3: v4stream, 4: anonymous")
	flag.StringVar(&C.S3cdn.Endpoint, "s3endpoint", "", "S3 接口")
	flag.StringVar(&C.S3cdn.Region, "s3region", "", "S3 区域")
	flag.StringVar(&C.S3cdn.Bucket, "s3bucket", "", "S3 存储桶")
	flag.StringVar(&C.S3cdn.RootDir, "s3rootdir", "", "S3 根目录")
	flag.StringVar(&C.S3cdn.Domain, "s3domain", "", "S3 CDN 域名")
	flag.Var(z.NewBoolVal(&C.S3cdn.Rewrite), "s3rewrite", "S3 是否覆盖")
	flag.StringVar(&C.S3cdn.AddrPort, "s3addrport", "0.0.0.0:88", "CND索引监听端口")
}

// 初始化方法， 处理 api 的而外配置接口
func Front2ServeByS3(api *front2.IndexApi, zgg *z.Zgg) {
	if !C.S3cdn.Enable {
		z.Println("[_cdnskip] s3cdn is disable")
		return
	}
	if C.S3cdn.Endpoint != "" {
		err := UploadToS3(api.HttpFS, api.FileFS, &api.Config, &C.S3cdn)
		if err != nil {
			zgg.ServeStop("upload to s3 error:", err.Error())
			return
		}
	}
	if C.S3cdn.Domain == "" || C.S3cdn.AddrPort == "" {
		zgg.ServeStop("run s3 serve config error: domain, addrport empty")
		return
	}
	// 提供 S3 索引服务
	zgg.Servers["(S3CDN)"] = &http.Server{Addr: C.S3cdn.AddrPort, //
		Handler: RunServeByS3(api.Config.Index, api.Config.Indexs, C.S3cdn.Domain, C.S3cdn.RootDir)}
}

// 上传到 S3
func UploadToS3(hfs http.FileSystem, fim map[string]fs.FileInfo, ffc *front2.Front2Config, cfg *S3cdnConfig) error {
	if cfg.Access == "" || cfg.Secret == "" || cfg.Region == "" || cfg.Bucket == "" || cfg.Domain == "" {
		return errors.New("config error: empty")
	}
	// https://github.com/minio/minio-go/

	ctx := context.Background()
	useSSL := false
	if zc.HasPrefixFold(cfg.Endpoint, "https://") {
		useSSL = true
		cfg.Endpoint = cfg.Endpoint[len("https://"):]
	} else if zc.HasPrefixFold(cfg.Endpoint, "http://") {
		cfg.Endpoint = cfg.Endpoint[len("http://"):]
	}
	if len(cfg.RootDir) > 0 && cfg.RootDir[0] == '/' {
		cfg.RootDir = cfg.RootDir[1:]
	}

	// Initialize minio client object.
	cli, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStatic(cfg.Access, cfg.Secret, cfg.TToken, credentials.SignatureType(cfg.Signer)),
		Region: cfg.Region,
		Secure: useSSL,
	})
	if err != nil {
		z.Println("[_cdnskip] minio client error:", err.Error())
		return err
	}

	exists, err := cli.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		z.Println("[_cdnskip] bucket exists error, [", cfg.Bucket, "]", err.Error())
		return err
	} else if !exists {
		// err = minioClient.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{Region: cfg.Region})
		return fmt.Errorf("bucket [%s] is not exists.", cfg.Bucket)
	}
	rootdir := filepath.Join(cfg.RootDir, z.AppName, z.Version)
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
		// 	z.Println("[_cdnskip] upload to s3:", name, ", exists")
		// 	return nil
		// }
		// if minio.ToErrorResponse(err).Code != "NoSuchKey" {
		// 	return err
		// }
		// z.Println("[_cdnskip] upload to s3:", err.Error(), ", noskip")
		// obj := <-cli.ListObjects(ctx, cfg.Bucket, minio.ListObjectsOptions{MaxKeys: 1, Prefix: rootdir})
		obj, err := cli.GetObject(ctx, cfg.Bucket, cnamepath, minio.GetObjectOptions{})
		if err == nil {
			bts, err := io.ReadAll(obj)
			if err != nil {
				if strings.ToLower(minio.ToErrorResponse(err).Code) != "nosuchkey" {
					z.Println("[_cdnskip] upload to s3:", cnamepath, ", read error:", err.Error())
					return err
				}
			} else {
				if string(bts) == cnametext {
					z.Println("[_cdnskip] upload to s3:", cnamepath, ", exists")
					return nil // skip
				} else {
					z.Println("[_cdnskip] upload to s3: cname no same", cnamepath, cnametext, string(bts))
				}
			}
		} else if strings.ToLower(minio.ToErrorResponse(err).Code) != "nosuchkey" {
			z.Println("[_cdnskip] upload to s3:", cnamepath, ", get object error:", err.Error())
			return err
		}
		z.Println("[_cdnskip] upload to s3:", cnamepath, ", noskip")
	}
	cnamebyte := bytes.NewReader([]byte(cnametext))
	cnamesize := cnamebyte.Size()
	if _, err = cli.PutObject(ctx, cfg.Bucket, cnamepath, cnamebyte, cnamesize, minio.PutObjectOptions{ContentType: "text/plain"}); err != nil {
		return err // panic(err) 上传标记域名文件
	}
	z.Println("[_success] upload to s3:", cnamepath)
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
	hfs http.FileSystem, fim map[string]fs.FileInfo, ffc *front2.Front2Config, cfg *S3cdnConfig) error {
	file, err := hfs.Open("/" + fpath)
	if err != nil {
		z.Println("[_cdn_put_] upload to s3:", fpath, ", open error:", err.Error())
		return err
	}
	defer file.Close()
	var rbts io.Reader
	var size int64
	isrp := false
	if front2.IsFixFile(fstat.Name(), ffc) {
		tbts, err := front2.GetFixFile(file, fstat.Name(), ffc.TmplRoot, rootpath, fim)
		if err != nil {
			z.Println("[_cdn_put_] upload to s3:", fpath, ", read error:", err.Error())
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
		z.Println("[_cdn_put_] upload to s3:", fpath, ", put error:", err.Error())
		return err
	}
	if isrp {
		z.Println("[_replace] upload to s3:", objname)
	} else {
		z.Println("[_success] upload to s3:", objname)
	}
	return nil
}

// 提供 S3 索引服务
func RunServeByS3(index string, indexs map[string]string, domain, rootdir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := index
		if ext := filepath.Ext(r.URL.Path); ext != "" {
			path = r.URL.Path
		} else {
			for k, v := range indexs {
				if r.URL.Path == k || zc.HasPrefixFold(r.URL.Path, k+"/") {
					path = v // 匹配到, 使用 v 代替 index
					break
				}
			}
		}
		path = domain + "/" + filepath.Join(rootdir, z.AppName, z.Version, path)
		resp, err := http.Get(path)
		if err != nil {
			z.Println("[_s3serve_] error, redirect to:", path, err.Error())
			http.Redirect(w, r, path, http.StatusMovedPermanently)
		} else {
			if ctype := resp.Header.Get("Content-Type"); strings.HasPrefix(ctype, "application/octet-stream") {
				resp.Header.Set("Content-Type", "text/html; charset=utf-8")
			}
			maps.Copy(w.Header(), resp.Header)
			w.WriteHeader(resp.StatusCode)
			io.Copy(w, resp.Body)
		}
	}
}
