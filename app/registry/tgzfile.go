package registry

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func CreateTgzFile(srcDir, outTgz string) error {
	ff, err := os.Create(outTgz)
	if err != nil {
		return err
	}
	defer ff.Close()
	return CreateTgzByWriter(srcDir, ff)
}

// 把 srcDir 的内容递归写入 outTgz （流式，不会把全部数据放到内存）
func CreateTgzByWriter(srcDir string, writer io.Writer) error {
	gw := gzip.NewWriter(writer)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// tar 中存储相对路径
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		// skip root (".") entry if you like; but we can include directories too
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}

		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = rel
		// 可选择设置 hdr.ModTime = time.Now()

		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}

		if info.Mode().IsRegular() {
			r, err := os.Open(path)
			if err != nil {
				return err
			}
			_, err = io.Copy(tw, r)
			_ = r.Close()
			return err
		}
		// 目录 / symlink 等无需写内容
		return nil
	})
}

func ExtractTgzFile(srcTgz, preDir, outDir string) error {
	ff, err := os.Open(srcTgz)
	if err != nil {
		return err
	}
	defer ff.Close()
	return ExtractTgzByReader(outDir, preDir, ff)
}

func ExtractTgzByReader(outDir, preDir string, reader io.Reader) error {
	gr, err := gzip.NewReader(reader)
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		name := hdr.Name
		if preDir != "" {
			if !strings.HasPrefix(name, preDir) {
				continue
			}
			name = name[len(preDir):]

		}
		target, err := SafeJoin(outDir, name)
		if err != nil {
			return err
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(hdr.Mode)); err != nil {
				return err
			}
		case tar.TypeReg: //, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				_ = out.Close()
				return err
			}
			_ = out.Close()
		case tar.TypeSymlink:
			// 创建符号链接（注意：linkname 要安全检查）
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			if err := os.Symlink(hdr.Linkname, target); err != nil {
				return err
			}
		default:
			// 对其它类型可按需处理
		}
	}
	return nil
}

// 防止路径穿越：确保目标路径在 outDir 下
func SafeJoin(outDir, entryName string) (string, error) {
	cleanName := filepath.ToSlash(filepath.Clean(entryName))
	// 去掉前导 /
	cleanName = strings.TrimPrefix(cleanName, "/")
	targetPath := filepath.Join(outDir, filepath.FromSlash(cleanName))
	absDest, _ := filepath.Abs(outDir)
	absTarget, _ := filepath.Abs(targetPath)
	if !strings.HasPrefix(absTarget, absDest+string(os.PathSeparator)) && absTarget != absDest {
		return "", os.ErrPermission
	}
	return targetPath, nil
}
