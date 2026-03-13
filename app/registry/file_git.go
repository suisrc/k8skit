package registry

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/memory"
)

// 通过网络获取 tgz 文件并解压， 支持 git 下载 tar.gz 归档文件 or http 下载 tgz 文件
// git 格式为： git+http(s)://username:password@github.com/username/repo.git#tagName
// tgz 格式为： http(s)://github.com/username/repo/archive/tagName.tar.gz
func GetFilesByGitOrTgz(outDir, preDir, rawURL string) error {
	if strings.HasPrefix(rawURL, "git+") {
		return GetFilesByGit(outDir, preDir, rawURL[4:])
	}
	return ExtractTgzByHttp(outDir, preDir, rawURL)
}

// 通过 git 创建 tgz 文件
func CreateTgzFileByGit(outFile, srcDir, preDir, rawURL string) error {
	opener, err := GetOpenerByGit(srcDir, preDir, rawURL)
	if err != nil {
		return err
	}
	writer, err := os.OpenFile(outFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer writer.Close()
	reader, err := opener()
	if err != nil {
		return err
	}
	defer reader.Close()
	_, err = io.Copy(writer, reader)
	return err
}

// ----------------------------------------------------------------------------------------

func GetFilesByGit(outDir, preDir, rawURL string) error {
	// 获取树
	tree, err := GetGitTree(rawURL)
	if err != nil {
		return err
	}
	// defer tree.Files().Close()
	// 遍历树，下载文件， 写入到 outDir 中， 使用 preDir 过滤
	err = tree.Files().ForEach(func(file *object.File) error {
		name := file.Name
		if preDir != "" {
			if !strings.HasPrefix(name, preDir) {
				return nil
			}
			name = name[len(preDir):]

		}
		target, err := SafeJoin(outDir, name)
		if err != nil {
			return err
		}
		fileDir := filepath.Dir(target)
		if _, err := os.Stat(fileDir); os.IsNotExist(err) {
			// 如果文件夹不存在，创建文件夹
			if err := os.MkdirAll(fileDir, 0644); err != nil {
				return err
			}
		}
		// 流写出文件
		fileReader, err := file.Reader()
		if err != nil {
			return err
		}
		defer fileReader.Close()
		// 创建文件
		fileFlag := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
		fileMode, err := file.Mode.ToOSFileMode()
		if err != nil {
			fileMode = 0644 // 读取失败时使用默认权限
		}
		fileWriter, err := os.OpenFile(target, fileFlag, fileMode)
		if err != nil {
			return err
		}
		defer fileWriter.Close()
		// 写入文件
		_, err = io.Copy(fileWriter, fileReader)
		return err
	})
	return err
}

// 通过 git 获取 tgz 文件， outDir 是 tgz 中的相对地址， preDir 是 git 中的相对地址
func GetOpenerByGit(outDir, preDir, rawURL string) (func() (io.ReadCloser, error), error) {
	// 获取树
	tree, err := GetGitTree(rawURL)
	if err != nil {
		return nil, err
	}
	return func() (io.ReadCloser, error) {
		// 使用管道实现流式输出
		pr, pw := io.Pipe()
		go func() {
			gw := gzip.NewWriter(pw)
			tw := tar.NewWriter(gw)
			err := tree.Files().ForEach(func(file *object.File) error {
				name := file.Name
				if preDir != "" {
					if !strings.HasPrefix(name, preDir) {
						return nil
					}
					name = name[len(preDir):]
				}
				// tar 中存储相对路径
				rel, err := filepath.Rel(outDir, name)
				if err != nil {
					return err
				}
				// skip root (".") entry if you like; but we can include directories too
				rel = filepath.ToSlash(rel)
				if rel == "." {
					return nil
				}
				// 获取文件内容流
				fileReader, err := file.Reader()
				if err != nil {
					return fmt.Errorf("read file %s error, %s", file.Name, err.Error())
				}
				defer fileReader.Close()
				// 构造tar头信息
				fileMode, err := file.Mode.ToOSFileMode()
				if err != nil {
					fileMode = 0644 // 读取失败时使用默认权限
				}
				header := &tar.Header{
					Name:    rel,
					Size:    file.Size,
					Mode:    int64(fileMode),
					ModTime: time.Now(),
				}
				// 写入头信息
				if err := tw.WriteHeader(header); err != nil {
					return fmt.Errorf("write tar header %s error, %s", file.Name, err.Error())
				}
				// 写入文件内容
				if _, err := io.Copy(tw, fileReader); err != nil {
					return fmt.Errorf("write file content %s error, %s", file.Name, err.Error())
				}
				return nil
			})
			if err != nil {
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
	}, nil
}

// 通过 git 获取 tgz 文件， outDir 是 tgz 中的相对地址， preDir 是 git 中的相对地址
func GetGitTree(rawURL string) (*object.Tree, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	tagName := parsedURL.Fragment
	if tagName == "" {
		return nil, errors.New("parser url error, no tag name")
	}
	// 提取用户基本信息
	var auth transport.AuthMethod = nil
	if username := parsedURL.User.Username(); username != "" {
		if password, exist := parsedURL.User.Password(); exist {
			auth = &githttp.BasicAuth{
				Username: username,
				Password: password,
			}
		} else {
			auth = &githttp.BasicAuth{
				Username: "oauth2",
				Password: username,
			}
		}
	}
	parsedURL.User = nil    // 删除用户信息
	parsedURL.Fragment = "" // 删除标签信息
	// 内存浅克隆指定标签，仅拉取单个提交
	cloneOpts := &git.CloneOptions{
		URL:           parsedURL.String(),
		ReferenceName: plumbing.NewTagReferenceName(tagName),
		SingleBranch:  true,
		Depth:         1, // 浅克隆，只拉取标签对应的提交，不下载历史
		Auth:          auth,
		NoCheckout:    false,
	}
	// 克隆到内存存储，不写磁盘
	repo, err := git.Clone(memory.NewStorage(), nil, cloneOpts)
	if err != nil {
		return nil, errors.New("git clone error, " + err.Error())
	}
	// 获取标签对应的提交树
	head, err := repo.Head()
	if err != nil {
		return nil, errors.New("get head error, " + err.Error())
	}
	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return nil, errors.New("get commit error, " + err.Error())
	}
	tree, err := commit.Tree()
	if err != nil {
		return nil, errors.New("get tree error, " + err.Error())
	}
	return tree, nil
}
