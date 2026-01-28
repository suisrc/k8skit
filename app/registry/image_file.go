package registry

import (
	"archive/tar"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
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

// 提取镜像中的文件， PS: 注意，最好只提取小文件，尽量不要提取大镜像
func ExtractImageFile(cfg *Config) error {
	if len(cfg.SrcPath) > 0 && cfg.SrcPath[0] == '/' {
		cfg.SrcPath = cfg.SrcPath[1:]
	}
	if cfg.Version == "" {
		if idx := strings.LastIndexByte(cfg.Image, ':'); idx > 0 {
			cfg.Version = cfg.Image[idx+1:]
		} else {
			cfg.Version = "latest"
		}
	}
	if cfg.Disable || cfg.Image == "" || cfg.OutPath == "" || cfg.OutPath == "none" {
		return nil
	}
	// 拉取镜像 remote.WithAuthFromKeychain(authn.DefaultKeychain)
	auz := authn.Anonymous // 匿名访问
	if cfg.Username != "" {
		auz = authn.FromConfig(authn.AuthConfig{Username: cfg.Username, Password: cfg.Password})
	} else if cfg.DcrAuths != "" {
		aus := map[string]map[string]authn.AuthConfig{}
		if err := json.Unmarshal([]byte(cfg.DcrAuths), &aus); err != nil {
			return errors.New("parse dcrauths: " + err.Error())
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
		return errors.New("parse image reference: " + err.Error())
	}
	z.Printf("[registry]: fetching image %s\n", ref.Name())
	// 拉取镜像
	img, err := remote.Image(ref, remote.WithAuth(auz))
	if err != nil {
		return errors.New("fetch image: " + err.Error())
	}
	layers, err := img.Layers()
	if err != nil {
		return errors.New("get layers: " + err.Error())
	}
	outAbs, err := filepath.Abs(cfg.OutPath)
	if err != nil {
		return errors.New("abs outpath: " + err.Error())
	}
	// 按顺序解层（layers 返回的顺序是从 base 到 top）
	lkey := ref.Name()
	if idx := strings.LastIndexByte(lkey, '/'); idx > 0 {
		lkey = lkey[idx+1:]
	}
	z.Printf("[registry]: (%s) fetching layers %d | %s -> %s\n", lkey, len(layers), cfg.SrcPath, cfg.OutPath)
	for i, layer := range layers {
		rc, err := layer.Uncompressed()
		if err != nil {
			return fmt.Errorf("layer[%d] uncompressed: %v", i, err)
		}
		if err := applyTarTarget(rc, outAbs, cfg.SrcPath); err != nil {
			rc.Close()
			return fmt.Errorf("apply layer[%d]: %v", i, err)
		}
		rc.Close()
		z.Printf("[registry]: (%s) applied layer %d/%d\n", lkey, i+1, len(layers))
	}
	z.Printf("[registry]: completed done %s\n", ref.Name())

	return nil
}

// applyTarOnlyTarget 将一个 tar（已解压，即 uncompressed layer stream）应用到 outDir，但只处理位于 srcPath 下的条目（例如 "www"）。
// 处理 whiteout 文件：.wh.<name> 表示删除同目录下 name；.wh..wh..opq 表示 opaque（删除该目录下所有先前内容）
func applyTarTarget(r io.Reader, outDir string, srcPath string) error {
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("reading tar: %w", err)
		}

		// sanitize path and ensure it stays inside outDir
		cleanName := filepath.Clean(hdr.Name)
		// strip leading "/" to avoid absolute paths
		cleanName = strings.TrimPrefix(cleanName, string(os.PathSeparator))
		// skip empty entries or "."
		if cleanName == "." || cleanName == "" {
			continue
		}

		// We only care about entries under srcPath, e.g. "www" or "www/..."
		// Normalize: allow entries that are exactly "www" or start with "www/"
		if cleanName != srcPath && !strings.HasPrefix(cleanName, srcPath+"/") {
			// 跳过非 /www 下的内容（包括 whiteout）
			continue
		}

		dest := filepath.Join(outDir, strings.TrimPrefix(cleanName, srcPath))

		// Ensure dest is inside outDir (防止 path traversal)
		if !strings.HasPrefix(dest, outDir+string(os.PathSeparator)) && dest != outDir {
			return fmt.Errorf("tar contains path outside output dir: %s", hdr.Name)
		}

		base := filepath.Base(cleanName)
		dir := filepath.Dir(cleanName)

		// whiteout handling (whiteout entries are also under srcPath because of the filter above)
		if strings.HasPrefix(base, ".wh.") {
			// opaque directory
			if base == ".wh..wh..opq" {
				targetDir := filepath.Join(outDir, dir)
				if err := removeAllChildren(targetDir); err != nil {
					return fmt.Errorf("apply opaque whiteout %s: %w", hdr.Name, err)
				}
				continue
			}
			// normal whiteout, remove the corresponding entry in lower layers
			targetName := strings.TrimPrefix(base, ".wh.")
			toRemove := filepath.Join(outDir, dir, targetName)
			if err := os.RemoveAll(toRemove); err != nil {
				// Treat remove failure as error (could be non-fatal depending on desired behavior)
				return fmt.Errorf("remove whiteout target %s: %w", toRemove, err)
			}
			continue
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(dest, fs.FileMode(hdr.Mode)); err != nil {
				return fmt.Errorf("mkdir %s: %w", dest, err)
			}
		case tar.TypeReg:
			// Ensure parent dir exists
			if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
				return fmt.Errorf("mkdir parent for %s: %w", dest, err)
			}
			// Create file (overwrite if exists)
			f, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, fs.FileMode(hdr.Mode))
			if err != nil {
				return fmt.Errorf("create file %s: %w", dest, err)
			}
			if _, err := io.CopyN(f, tr, hdr.Size); err != nil && err != io.EOF {
				f.Close()
				return fmt.Errorf("write file %s: %w", dest, err)
			}
			f.Close()
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
				return fmt.Errorf("mkdir parent for symlink %s: %w", dest, err)
			}
			_ = os.Remove(dest)
			// Note: symlink targets may be absolute or relative; do not rewrite them here.
			if err := os.Symlink(hdr.Linkname, dest); err != nil {
				return fmt.Errorf("symlink %s -> %s: %w", dest, hdr.Linkname, err)
			}
		case tar.TypeLink:
			// hard link: only handle if target is also inside srcPath
			linkTarget := filepath.Clean(strings.TrimPrefix(hdr.Linkname, string(os.PathSeparator)))
			if linkTarget != srcPath && !strings.HasPrefix(linkTarget, srcPath+"/") {
				// target is outside of srcPath; skip or error (we skip)
				continue
			}
			target := filepath.Join(outDir, linkTarget)
			// remove existing and create link if target exists
			_ = os.Remove(dest)
			if err := os.Link(target, dest); err != nil {
				return fmt.Errorf("hardlink %s -> %s: %w", dest, target, err)
			}
		case tar.TypeChar, tar.TypeBlock, tar.TypeFifo:
			// skip special device files for safety
			continue
		default:
			// ignore other types
			continue
		}
	}
}

// removeAllChildren removes all children of dir, but keeps the dir itself.
func removeAllChildren(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		// if dir doesn't exist that's fine
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		p := filepath.Join(dir, e.Name())
		if err := os.RemoveAll(p); err != nil {
			return err
		}
	}
	return nil
}
