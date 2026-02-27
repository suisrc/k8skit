package s3cdn

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/suisrc/zgg/app/front2"
	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/zc"
)

var (
	C = struct {
		S3cdn Config
	}{}
)

type Config struct {
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
	z.RegKey(zgg.SvcKit, false, "front2", api) // 注入服务
	if !C.S3cdn.Enable {
		z.Println("[_cdnskip]: s3cdn is disable", zc.CFG_ENV+"_S3CDN_ENABLE=false")
		return
	}
	if C.S3cdn.Endpoint != "" {
		err := UploadToS3(api.HttpFS, api.FileFS, &api.Config, &C.S3cdn, z.AppName, z.Version)
		if err != nil {
			zgg.ServeStop("upload to s3 error:", err.Error())
			return
		}
	}
	if C.S3cdn.Domain == "" || C.S3cdn.AddrPort == "" {
		zgg.ServeStop("run s3 serve config error: domain, addrport empty")
		return
	}
	// c创建 S3 CDN 应答桥接服务
	hdl := front2.NewApi(nil, api.Config, "[_s3serve]")
	InitCdnServe(hdl, C.S3cdn.Domain, C.S3cdn.RootDir, z.AppName, z.Version)
	zgg.Servers["(S3CDN)"] = &http.Server{Addr: C.S3cdn.AddrPort, Handler: hdl}
}

// 获取访问客户端
func GetClient(ctx context.Context, cfg *Config) (*minio.Client, error) {
	if cfg.Access == "" || cfg.Secret == "" || cfg.Region == "" || cfg.Bucket == "" || cfg.Domain == "" {
		return nil, errors.New("config error: empty")
	}
	// https://github.com/minio/minio-go/

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
		z.Println("[_cdnskip]: minio client error:", err.Error())
		return nil, err
	}

	exists, err := cli.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		z.Println("[_cdnskip]: bucket exists error, [", cfg.Bucket, "]", err.Error())
		return nil, err
	} else if !exists {
		// err = minioClient.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{Region: cfg.Region})
		return nil, fmt.Errorf("bucket [%s] is not exists.", cfg.Bucket)
	}

	return cli, nil
}
