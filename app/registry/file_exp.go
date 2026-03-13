package registry

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing/object"
)

// 其中的方法未经大量测试，不建议使用， 实验性方法

type OpenerFileAddFn func(fileReader io.ReadCloser, fileName string, fileSize, fileMode int64, fileTime time.Time) error

func GetOpenerByPipe(outDir, preDir string, fileEachFn func(fileAddFn OpenerFileAddFn) error) func() (io.ReadCloser, error) {
	return func() (io.ReadCloser, error) {
		// 使用管道实现流式输出
		pr, pw := io.Pipe()
		go func() {
			gw := gzip.NewWriter(pw)
			tw := tar.NewWriter(gw)
			fn := func(fileReader io.ReadCloser, fileName string, fileSize, fileMode int64, fileTime time.Time) error {
				if fileReader == nil {
					return nil // 遍历结束
				}
				if preDir != "" {
					if !strings.HasPrefix(fileName, preDir) {
						return nil
					}
					fileName = fileName[len(preDir):]
				}
				// tar 中存储相对路径
				rel, err := filepath.Rel(outDir, fileName)
				if err != nil {
					return err
				}
				// skip root (".") entry if you like; but we can include directories too
				rel = filepath.ToSlash(rel)
				if rel == "." {
					return nil
				}
				// 构造tar头信息
				header := &tar.Header{
					Name:    rel,
					Size:    fileSize,
					Mode:    fileMode,
					ModTime: fileTime,
				}
				// 写入头信息
				if err := tw.WriteHeader(header); err != nil {
					return fmt.Errorf("write tar header %s error, %s", fileName, err.Error())
				}
				// 写入文件内容
				if _, err := io.Copy(tw, fileReader); err != nil {
					return fmt.Errorf("write file content %s error, %s", fileName, err.Error())
				}
				return nil
			}
			// 遍历文件, 执行文件处理函数
			if err := fileEachFn(fn); err != nil {
				_ = pw.CloseWithError(err)
				return
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

// 同 GetOpenerByFs
func GetOpenerByFs0(ffs []string, src, out string) func() (io.ReadCloser, error) {
	return GetOpenerByPipe(out, "", func(fileAddFn OpenerFileAddFn) error {
		fileAddFn2 := func(srcPath string) error {
			ff, err := os.Open(srcPath)
			if err != nil {
				return err
			}
			defer ff.Close()
			fi, err := ff.Stat()
			if err != nil {
				return err
			}
			return fileAddFn(ff, srcPath, fi.Size(), int64(fi.Mode()), fi.ModTime())
		}
		for _, ff := range ffs {
			if ff == src {
				// 直接返回结果
				return fileAddFn2(ff)
			} else if err := fileAddFn2(filepath.ToSlash(filepath.Join(src, ff))); err != nil {
				// 发生异常返回结果
				return err
			}
		}
		return nil
	})
}

// 同 GetOpenerByGit
func GetOpenerByGit0(outDir, preDir, rawURL string) (func() (io.ReadCloser, error), error) {
	// 获取树
	tree, err := GetGitTree(rawURL)
	if err != nil {
		return nil, err
	}
	return GetOpenerByPipe(outDir, preDir, func(fileAddFn OpenerFileAddFn) error {
		return tree.Files().ForEach(func(file *object.File) error {
			// 获取文件内容流
			fileReader, err := file.Reader()
			if err != nil {
				return fmt.Errorf("read file %s error, %s", file.Name, err.Error())
			}
			defer fileReader.Close()
			fileMode, err := file.Mode.ToOSFileMode()
			if err != nil {
				fileMode = 0644 // 读取失败时使用默认权限
			}
			return fileAddFn(fileReader, file.Name, file.Size, int64(fileMode), time.Now())
		})
	}), nil
}
