package s3cdn

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"maps"
	"mime"
	"net/http"
	"path/filepath"
	"slices"
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
	Access   string `json:"access"`   // 账号
	Secret   string `json:"secret"`   // 秘钥
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

	flag.StringVar(&C.S3cdn.Access, "s3access", "", "S3 账号")
	flag.StringVar(&C.S3cdn.Secret, "s3secret", "", "S3 秘钥")
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
	if C.S3cdn.Endpoint != "" {
		err := UploadToS3(api, zgg, C.S3cdn)
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
func UploadToS3(api *front2.IndexApi, zgg *z.Zgg, cfg S3cdnConfig) error {
	if cfg.Access == "" || cfg.Secret == "" || cfg.Region == "" || cfg.Bucket == "" || cfg.Domain == "" {
		zgg.ServeStop("s3cdn config error: empty")
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

	// Initialize minio client object.
	cli, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.Access, cfg.Secret, ""),
		Region: cfg.Region,
		Secure: useSSL,
	})
	if err != nil {
		return err
	}

	exists, err := cli.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return err
	} else if !exists {
		// err = minioClient.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{Region: cfg.Region})
		return fmt.Errorf("bucket [%s] is not exists.", cfg.Bucket)
	}
	rootdir := filepath.Join(cfg.RootDir, z.AppName, z.Version)
	if !cfg.Rewrite {
		// 判断 index 文件是否存在，如果存在，跳过上传
		name := filepath.Join(rootdir, api.Config.Index)
		_, err = cli.StatObject(ctx, cfg.Bucket, name, minio.StatObjectOptions{})
		if err == nil {
			z.Println("[_cdnskip_] upload to s3:", name, ", exists")
			return nil
		}
		if minio.ToErrorResponse(err).Code != "NoSuchKey" {
			return err
		}
		z.Println("[_cdnskip_] upload to s3:", err.Error(), ", noskip")
		// lst := cli.ListObjects(ctx, cfg.Bucket, minio.ListObjectsOptions{MaxKeys: 1, Prefix: rootdir})
		// obj := <-lst
		// if obj.Err == nil {
		// 	z.Println("[_cdnskip_] upload to s3:", obj.Key, ", exists")
		// 	return nil
		// }
	}
	// if err != nil && minio.ToErrorResponse(err).Code == "NoSuchKey"

	// 遍历 api 中所有的文件夹执行上传
	return UploadToS3Loop(ctx, cli, cfg.Bucket, rootdir, cfg.Domain, api.HttpFS, //
		api.Config.Folder, api.Config.Folder, api.Config.TmplPath, api.Config.TmplSuff)
}

// 循环上传文件
func UploadToS3Loop(ctx context.Context, cli *minio.Client, bucket, rootdir, domain string, httpfs http.FileSystem, //
	root, fpath, tpath string, exts []string) error {

	rp := domain + "/" + rootdir // 静态资源访问真实路径
	if strings.HasPrefix(rp, "https:") {
		rp = rp[6:]
	} else if strings.HasPrefix(rp, "http:") {
		rp = rp[5:]
	} else if !strings.HasPrefix(rp, "//") {
		rp = "//" + rp
	}
	tp := tpath // 文件中的模版路径
	if tp != "" && !strings.HasPrefix(tp, "/") {
		tp = "/" + tp
	}
	z.Println("[_replace] upload to s3:", tp, "->", rp)
	rpbs := []byte(rp)
	tpbs := []byte(tp)

	// 循环遍历文件， 比递归遍历性能更高
	paths := []string{fpath}
	n := len(paths)
	for n > 0 {
		n -= 1
		last := paths[n]
		paths = paths[:n]

		if hfs, err := httpfs.Open(last); err != nil {
			return err // 打开文件失败
		} else if stat, err := hfs.Stat(); err != nil {
			hfs.Close()
			return err // 获取文件信息失败
		} else if stat.IsDir() {
			// 是文件夹， 变量所有文件，加入要遍历的路径列表中
			dir, err := hfs.Readdir(-1)
			hfs.Close()
			if err != nil {
				return err
			}
			for _, hst := range dir {
				paths = append(paths, filepath.Join(last, hst.Name()))
			}
			n = len(paths)
		} else {
			// 是文件， 上传文件
			name := filepath.Join(rootdir, strings.TrimPrefix(last, root))
			ext := filepath.Ext(last)
			var rbts io.Reader
			var size int64
			// tp == "", 全局跳过替换
			isrp := tp != "" && slices.Contains(exts, ext)
			if isrp {
				// /ROOT_PATH -> rp， 需要调整文件中对于资源的方位路径
				tbts, _ := io.ReadAll(hfs)
				tbts = bytes.ReplaceAll(tbts, tpbs, rpbs)
				rbts = bytes.NewReader(tbts)
				size = int64(len(tbts))
			} else {
				rbts = hfs
				size = stat.Size()
			}
			ctype := mime.TypeByExtension(ext) // 获取文件类型
			// 上传对象
			_, err := cli.PutObject(ctx, bucket, name, rbts, size, minio.PutObjectOptions{ContentType: ctype})
			hfs.Close()
			if err != nil {
				// panic(err)
				return err
			}
			if isrp {
				z.Println("[_replace] upload to s3:", name)
			} else {
				z.Println("[_success] upload to s3:", name)
			}
		}
	}
	return nil
}

// 提供 S3 索引服务
func RunServeByS3(index string, indexs map[string]string, domain, rootdir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		next := index
		for k, v := range indexs {
			if zc.HasPrefixFold(path, k) {
				// 匹配到, 使用 v 代替 index
				next = v
				break
			}
		}
		next = domain + "/" + filepath.Join(rootdir, z.AppName, z.Version, next)
		resp, err := http.Get(next)
		if err != nil {
			z.Println("[_s3serve_] error, redirect to:", next, err.Error())
			http.Redirect(w, r, next, http.StatusMovedPermanently)
		} else {
			maps.Copy(w.Header(), resp.Header)
			w.WriteHeader(resp.StatusCode)
			io.Copy(w, resp.Body)
		}
	}
}
