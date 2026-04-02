package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/suisrc/zgg/z"
)

type Config struct {
	Disable  bool   `json:"disable"`  // 禁用
	Username string `json:"username"` // 用户, 为空时匿名访问
	Password string `json:"password"` // 密码
	Image    string `json:"image"`    // 镜像
	SrcPath  string `json:"srcpath"`  // 源路径
	OutPath  string `json:"outpath"`  // 目标路径
	Version  string `json:"version"`  // 版本
	DcrAuths string `json:"dcrauths"` // 认证， {"auths":{"exp.com":{"username":"user","password":"pass"}}}
}

// 拉取镜像
func PullImpage(cfg *Config) (v1.Image, error) {
	// 拉取镜像 remote.WithAuthFromKeychain(authn.DefaultKeychain)
	auz := authn.Anonymous // 匿名访问
	if cfg.Username != "" {
		auz = authn.FromConfig(authn.AuthConfig{Username: cfg.Username, Password: cfg.Password})
	} else if cfg.DcrAuths != "" {
		aus := map[string]map[string]authn.AuthConfig{}
		if err := json.Unmarshal([]byte(cfg.DcrAuths), &aus); err != nil {
			return nil, errors.New("parse dcrauths: " + err.Error())
		}
		hcr := ""
		if idx := strings.IndexByte(cfg.Image, '/'); idx <= 0 {
			hcr = "docker.io" // 基础镜像仓库
		} else if cdx := strings.IndexByte(cfg.Image[:idx], '.'); cdx > 0 {
			hcr = cfg.Image[:idx] // 镜像仓库
		}
		if auths, ok := aus["auths"]; !ok {
			// pass, 没有匹配的，使用匿名访问
		} else if hcr == "" && len(auths) > 0 {
			// 取第一个仓库， 并且补充镜像仓库地址
			for key, auth := range auths {
				cfg.Image = filepath.Join(key, cfg.Image)
				auz = authn.FromConfig(auth)
				break // 只取第一条
			}
		} else if auth, ok := auths[hcr]; ok {
			// 通过域名获取访问令牌
			auz = authn.FromConfig(auth)
			// z.Logn(z.ToStr(auth))
		} // else 没有匹配的，使用匿名访问
	}
	// 解析镜像
	ref, err := name.ParseReference(cfg.Image)
	if err != nil {
		return nil, errors.New("parse image reference: " + err.Error())
	}
	z.Logf("[registry]: fetching image %s\n", ref.Name())
	// 拉取镜像
	img, err := remote.Image(ref, remote.WithAuth(auz))
	if err != nil {
		return nil, errors.New("fetch image: " + err.Error())
	}
	return img, nil
}

// 推送镜像
func PushImage(tag name.Tag, img v1.Image, cfg *Config) error {
	hcr := tag.Registry.Name()
	if hcr == "" {
		return errors.New("no registry")
	}
	var auz authn.Authenticator // 匿名访问
	if cfg.Username != "" {
		auz = authn.FromConfig(authn.AuthConfig{Username: cfg.Username, Password: cfg.Password})
	} else if cfg.DcrAuths != "" {
		aus := map[string]map[string]authn.AuthConfig{}
		if err := json.Unmarshal([]byte(cfg.DcrAuths), &aus); err != nil {
			return errors.New("parse dcrauths: " + err.Error())
		}
		hcr := tag.Registry.Name()
		if auths, ok := aus["auths"]; !ok {
			// pass, 没有匹配的
		} else if auth, ok := auths[hcr]; ok {
			auz = authn.FromConfig(auth)
		} // else 没有匹配的
	}
	if auz == nil {
		return errors.New("no authenticator")
	}
	// 推送镜像
	return remote.Write(tag, img, remote.WithAuth(auz))
}

// 向镜像中导入文件， 默认从本地目录导入， 然后上传到 Version 对应的镜像中
func ImportImage(cfg *Config, opr func() (io.ReadCloser, error), patch func(*v1.Image) error) error {
	if len(cfg.OutPath) > 0 && cfg.OutPath[0] == '/' {
		cfg.OutPath = cfg.OutPath[1:] //  容器内目标路径（相对于 /，不要以 / 开头）
	}
	if cfg.Disable || cfg.Image == "" || cfg.SrcPath == "" || cfg.SrcPath == "none" || cfg.Version == "" {
		return nil // 原始路径不存在或者推送版本为空，终止
	}
	// 由于下载的镜像和推送的镜像可能不在同一个注册表中，所以，默认 Version 是 image:tag 整体， 如果没有Version，则从 Image 中获取
	newTagRef := cfg.Version
	if idx := strings.IndexByte(newTagRef, ':'); idx < 0 {
		// 只有版本， 则从 Image 中获取 Repo
		if idx := strings.IndexByte(cfg.Image, ':'); idx > 0 {
			newTagRef = cfg.Image[:idx+1] + cfg.Version
		} else {
			newTagRef = cfg.Image + ":" + cfg.Version
		}
	}
	newTag, err := name.NewTag(newTagRef)
	if err != nil {
		return errors.New("parse tag error: " + err.Error())
	}
	// 获取导入的文件句柄
	if opr == nil {
		if opener, err := GetOpenerToLayer(cfg.SrcPath, cfg.OutPath, ""); err != nil {
			return errors.New("get layer opener error: " + err.Error())
		} else {
			opr = opener
		}
	}
	// 下载文件
	baseImg, err := PullImpage(cfg)
	if err != nil {
		return errors.New("pull image error: " + err.Error())
	}
	// 使用 LayerFromOpener（流式，不会一次性将全部数据读入内存）
	layer, err := tarball.LayerFromOpener(opr)
	if err != nil {
		return errors.New("form opener build layer fail: " + err.Error())
	}
	// 关键行（追加 layer）
	newImg, err := mutate.AppendLayers(baseImg, layer)
	if err != nil {
		return errors.New("append layer fail: " + err.Error())
	}
	if patch != nil {
		if err := patch(&newImg); err != nil {
			return errors.New("patch image fail: " + err.Error())
		}
	}
	// 读取现有的 ConfigFile，修改 Cmd / Entrypoint，然后应用, 可通过 patch 修改
	// cfgFile, err := newImg.ConfigFile()
	// if err != nil {
	// 	return errors.New("get config file fail: " + err.Error())
	// }
	// // 修改容器启动命令（示例：运行 /app/run.sh）
	// cfgFile.Config.Entrypoint = []string{"/bin/sh", "-c"}
	// cfgFile.Config.Cmd = []string{"/app/run.sh"}
	// // 应用新的 config
	// newImg, err := mutate.ConfigFile(newImg, cfgFile)
	// if err != nil {
	// 	return errors.New("modify config fail: " + err.Error())
	// }
	// 将新镜像写到新的 tag（不覆盖原来的 tag）
	if err := PushImage(newTag, newImg, cfg); err != nil {
		return errors.New("push image fail: " + err.Error())
	}
	// 更新配置信息
	cfg.Version = newTag.TagStr() // 版本信息
	cfg.Image = newTag.Name()
	return nil
}

// 从镜像中导出文件， PS: 注意，最好只提取小文件，尽量不要提取大镜像
func ExportImage(cfg *Config) error {
	if len(cfg.SrcPath) > 0 && cfg.SrcPath[0] == '/' {
		cfg.SrcPath = cfg.SrcPath[1:]
	}
	if cfg.Version == "" {
		if idx := strings.IndexByte(cfg.Image, ':'); idx > 0 {
			cfg.Version = cfg.Image[idx+1:]
		} else {
			cfg.Version = "latest"
		}
	}
	if cfg.Disable || cfg.Image == "" || cfg.OutPath == "" || cfg.OutPath == "none" {
		return nil
	}
	// 下载文件
	img, err := PullImpage(cfg)
	if err != nil {
		return err
	}
	// ref, _ := name.ParseReference(cfg.Image)
	// 获取镜像的层， 从中导出文件
	layers, err := img.Layers()
	if err != nil {
		return errors.New("get layers: " + err.Error())
	}
	outAbs, err := filepath.Abs(cfg.OutPath)
	if err != nil {
		return errors.New("abs outpath: " + err.Error())
	}
	// 按顺序解层（layers 返回的顺序是从 base 到 top）
	key_ := cfg.Image
	if idx := strings.LastIndexByte(key_, '/'); idx > 0 {
		key_ = key_[idx+1:]
	}
	z.Logf("[registry]: (%s) fetching layers %d | %s -> %s\n", key_, len(layers), cfg.SrcPath, cfg.OutPath)
	for i, layer := range layers {
		rc, err := layer.Uncompressed()
		if err != nil {
			return fmt.Errorf("layer[%d] uncompressed: %v", i, err)
		}
		if err := ApplyTarToLayer(rc, outAbs, cfg.SrcPath); err != nil {
			rc.Close()
			return fmt.Errorf("apply layer[%d]: %v", i, err)
		}
		rc.Close()
		z.Logf("[registry]: (%s) applied layer %d/%d\n", key_, i+1, len(layers))
	}
	z.Logf("[registry]: completed done %s\n", cfg.Image)

	return nil
}
