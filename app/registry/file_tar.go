package registry

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	// "github.com/ulikunitz/xz"
)

// 返回 tarball.Opener 给 LayerFromOpener 使用, 返回是 tgz 格式
func GetOpenerToLayer(src, out, pre string) (func() (io.ReadCloser, error), error) {
	var ffs []string
	// 判断是否是一个文件
	if ffi, err := os.Stat(src); err != nil {
		return nil, errors.New("src path is not a file or directory: " + src)
	} else if ffi.IsDir() {
		// 获取导入的文件
		ffs, err := GetFileList(src)
		if err != nil {
			return nil, err
		} else if len(ffs) == 0 {
			return nil, errors.New("src path is empty: " + src)
		}
	} else if ext := filepath.Ext(src); ext == ".tar" || ext == ".gz" || ext == ".tgz" {
		// 解压文件并放入指定文件夹中
		return GetOpenerByTar(src, out, pre, ext), nil
	} else {
		// 单一文件
		ffs = append(ffs, src)
	}
	return GetOpenerByFs(ffs, src, out), nil
}

// 归档文件获取读出流, 返回的 io.ReadCloser 是 tgz 格式
// ffs 可以使用 GetFileList 获取
func GetOpenerByFs(ffs []string, src, out string) func() (io.ReadCloser, error) {
	return func() (io.ReadCloser, error) {
		pr, pw := io.Pipe()
		go func() {
			gw := gzip.NewWriter(pw)
			tw := tar.NewWriter(gw)
			for _, ff := range ffs {
				if ff == src {
					if err := AddFileToTar(tw, src, out); err != nil {
						_ = tw.Close()
						_ = gw.Close()
						_ = pw.CloseWithError(err)
						return
					}
					break // 文件对拷， 直接结束
				}
				hostPath := filepath.ToSlash(filepath.Join(src, ff))
				targetPath := filepath.ToSlash(filepath.Join(out, ff))
				if err := AddFileToTar(tw, hostPath, targetPath); err != nil {
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
		return pr, nil
	}
}

// 用于扩展 Tar 归档文件解析能力
var ExtTarReaders = map[string]func(*os.File) (*tar.Reader, error){
	// ".xz": func(f *os.File) (*tar.Reader, error) {
	// 	rdr, err := xz.NewReader(f)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	return tar.NewReader(rdr), nil
	// },
}

// 归档文件获取读出流, 特别注意： 未进行文件识别， 返回的 io.ReadCloser 是 tgz 格式
// 目前支持： .gz/.tgz, .tar(默认) 两种格式, 可通过 在 ExtTarReaders 中增加 reader 记性扩展
// 因为需要涉及到内部文件路径的转换，所以不要解压后再压缩, 如果 out， pre 为空，直接返回文件流
func GetOpenerByTar(src, out, pre, ext string) func() (io.ReadCloser, error) {
	return func() (io.ReadCloser, error) {
		if out == "" && pre == "" && (ext == ".gz" || ext == ".tgz") {
			return os.Open(src) // 直接读取文件，返回文件流即可
		}
		// opener 在被调用时创建 pipe 并在 goroutine 中写入 tgz 到 pipe writer
		pr, pw := io.Pipe()
		go func() {
			// 打开 archive(归档文件) 并根据扩展名选择解压器
			ff, err := os.Open(src)
			if err != nil {
				_ = pw.CloseWithError(err)
				return
			}
			defer ff.Close()
			var tr *tar.Reader
			// 选择解压器
			if fn, ok := ExtTarReaders[ext]; ok {
				// 使用扩展解压器
				tr, err = fn(ff)
				if err != nil {
					_ = pw.CloseWithError(err)
					return
				}
			} else {
				// 使用内嵌解压器
				switch ext {
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
				if pre != "" {
					if !strings.HasPrefix(name, pre) {
						continue // 跳过非指定路径下的文件
					} else {
						name = name[len(pre):]
					}
				}
				// 防止以 / 开头
				name = filepath.ToSlash(filepath.Join(out, name))
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
	}
}

// 将主机文件写入 tar.Writer（写入 header + 内容）
func AddFileToTar(tw *tar.Writer, srcPath, outPath string) error {
	ff, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer ff.Close()
	fi, err := ff.Stat()
	if err != nil {
		return err
	}
	hdr := &tar.Header{
		Name:    filepath.ToSlash(outPath),
		Mode:    int64(fi.Mode()),
		Size:    fi.Size(),
		ModTime: fi.ModTime(),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	_, err = io.Copy(tw, ff)
	return err
}

func GetFileList(root string) ([]string, error) {
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

// RemoveAllDir removes all children of dir, but keeps the dir itself.
func RemoveAllDir(dir string) error {
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

//-----------------------------------------------------------------------------------------------------------------------------

// ApplyTarTarget 将一个 tar（已解压，即 uncompressed layer stream）应用到 outDir，但只处理位于 srcPath 下的条目（例如 "www"）。
// 处理 whiteout 文件：.wh.<name> 表示删除同目录下 name；.wh..wh..opq 表示 opaque（删除该目录下所有先前内容）
// 这是注册表镜像专用的归档文件处理逻辑
func ApplyTarToLayer(r io.Reader, outDir string, srcPath string) error {
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
				if err := RemoveAllDir(targetDir); err != nil {
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
		case tar.TypeReg: //, tar.TypeRegA:
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
