package registry

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	// "github.com/ulikunitz/xz"
)

// 向镜像中导入文件， 默认从本地目录导入， 然后上传到 Version 对应的镜像中
func ImportFile(cfg *Config, opr tarball.Opener) error {
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
		if opener, err := getLayerOpener(cfg, ""); err != nil {
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

	// // 读取现有的 ConfigFile，修改 Cmd / Entrypoint，然后应用
	// cfgFile, err := newImg.ConfigFile()
	// if err != nil {
	// 	return errors.New("get config file fail: " + err.Error())
	// }
	// // 修改容器启动命令（示例：运行 /app/run.sh）
	// cfgFile.Config.Entrypoint = []string{"/bin/sh", "-c"}
	// cfgFile.Config.Cmd = []string{"/app/run.sh"}
	// // 应用新的 config
	// newImg2, err := mutate.ConfigFile(newImg, cfgFile)
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

func getLayerOpener(cfg *Config, pre string) (tarball.Opener, error) {
	var ffs []string
	// 判断是否是一个文件
	if ffi, err := os.Stat(cfg.SrcPath); err != nil {
		return nil, errors.New("src path is not a file or directory: " + cfg.SrcPath)
	} else if ffi.IsDir() {
		// 获取导入的文件
		ffs, err := getFileList(cfg.SrcPath)
		if err != nil {
			return nil, err
		} else if len(ffs) == 0 {
			return nil, errors.New("src path is empty: " + cfg.SrcPath)
		}
	} else if ext := filepath.Ext(cfg.SrcPath); ext == ".tar" || ext == ".gz" || ext == ".tgz" {
		// 解压文件并放入指定文件夹中
		return func() (io.ReadCloser, error) {
			pr, pw := io.Pipe()
			go func() {
				// 打开 archive 并根据扩展名选择解压器
				ff, err := os.Open(cfg.SrcPath)
				if err != nil {
					_ = pw.CloseWithError(err)
					return
				}
				defer ff.Close()
				var tr *tar.Reader
				// 选择解压器
				switch ext {
				// case ".xz":
				// 	xzr, err := xz.NewReader(f)
				// 	if err != nil {
				// 		_ = pw.CloseWithError(err)
				// 		return
				// 	}
				// 	tr = tar.NewReader(xzr)
				case ".gz", ".tgz":
					gzr, err := gzip.NewReader(ff)
					if err != nil {
						_ = pw.CloseWithError(err)
						return
					}
					defer gzr.Close()
					tr = tar.NewReader(gzr)
				default:
					// 假设是未压缩的 tar
					tr = tar.NewReader(ff)
				}
				// 写入 gzip+tar 到 pw
				gw := gzip.NewWriter(pw)
				tw := tar.NewWriter(gw)
				for {
					hdr, err := tr.Next()
					if err == io.EOF {
						break
					}
					if err != nil {
						_ = tw.Close()
						_ = gw.Close()
						_ = pw.CloseWithError(err)
						return
					}
					// 调整目标路径（加 prefix，去掉绝对路径或 ..）
					name := filepath.ToSlash(filepath.Clean(hdr.Name))
					if name == "." || name == "/" {
						continue
					}
					if pre != "" && !strings.HasPrefix(name, pre) {
						continue // 跳过非指定路径下的文件
					} else if pre != "" {
						name = strings.TrimPrefix(name, pre)
					}
					// 防止以 / 开头
					name = filepath.ToSlash(filepath.Join(cfg.OutPath, name))
					// 构造新的 header，注意移除 pax/global headers 中的字段以便兼容
					nh := &tar.Header{
						Name:    name,
						Mode:    hdr.Mode,
						Size:    hdr.Size,
						ModTime: hdr.ModTime,
					}
					if err := tw.WriteHeader(nh); err != nil {
						_ = tw.Close()
						_ = gw.Close()
						_ = pw.CloseWithError(err)
						return
					}
					if _, err := io.Copy(tw, tr); err != nil {
						_ = tw.Close()
						_ = gw.Close()
						_ = pw.CloseWithError(err)
						return
					}
				}
				// 关闭 writers
				if err := tw.Close(); err != nil {
					_ = pw.CloseWithError(err)
					return
				}
				if err := gw.Close(); err != nil {
					_ = pw.CloseWithError(err)
					return
				}
				_ = pw.Close()
			}()
			return pr, nil
		}, nil
	} else {
		ffs = append(ffs, cfg.SrcPath)
	}
	// opener 在被调用时创建 pipe 并在 goroutine 中写入 gzip+tar 到 pipe writer，
	// 返回的 reader 将被 tarball.LayerFromOpener 使用。
	return func() (io.ReadCloser, error) {
		pr, pw := io.Pipe()
		go func() {
			gw := gzip.NewWriter(pw)
			tw := tar.NewWriter(gw)
			for _, ff := range ffs {
				if ff == cfg.SrcPath {
					if err := addFileToTar(tw, cfg.SrcPath, cfg.OutPath); err != nil {
						_ = tw.Close()
						_ = gw.Close()
						_ = pw.CloseWithError(err)
						return
					}
					break // 文件对拷， 直接结束
				}
				hostPath := filepath.ToSlash(filepath.Join(cfg.SrcPath, ff))
				targetPath := filepath.ToSlash(filepath.Join(cfg.OutPath, ff))
				if err := addFileToTar(tw, hostPath, targetPath); err != nil {
					_ = tw.Close()
					_ = gw.Close()
					_ = pw.CloseWithError(err)
					return
				}
			}
			// 关闭 writers（顺序：tar -> gzip -> pipe writer）
			if err := tw.Close(); err != nil {
				_ = pw.CloseWithError(err)
				return
			}
			if err := gw.Close(); err != nil {
				_ = pw.CloseWithError(err)
				return
			}
			_ = pw.Close()
		}()
		// 返回 reader 给 LayerFromOpener 使用
		return pr, nil
	}, nil
}

// 将主机文件写入 tar.Writer（写入 header + 内容）
func addFileToTar(tw *tar.Writer, hostPath, targetPath string) error {
	ff, err := os.Open(hostPath)
	if err != nil {
		return err
	}
	defer ff.Close()

	info, err := ff.Stat()
	if err != nil {
		return err
	}

	hdr := &tar.Header{
		Name:    filepath.ToSlash(targetPath),
		Mode:    0644,
		Size:    info.Size(),
		ModTime: time.Now(),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	_, err = io.Copy(tw, ff)
	return err
}

func getFileList(root string) ([]string, error) {
	var ffs []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			ffs = append(ffs, path)
		}
		return nil
	})
	return ffs, err
}
