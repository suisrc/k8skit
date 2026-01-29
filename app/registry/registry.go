package registry

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
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
			// z.Println(z.ToStr(auth))
		} // else 没有匹配的，使用匿名访问
	}
	// 解析镜像
	ref, err := name.ParseReference(cfg.Image)
	if err != nil {
		return nil, errors.New("parse image reference: " + err.Error())
	}
	z.Printf("[registry]: fetching image %s\n", ref.Name())
	// 拉取镜像
	img, err := remote.Image(ref, remote.WithAuth(auz))
	if err != nil {
		return nil, errors.New("fetch image: " + err.Error())
	}
	return img, nil
}

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
