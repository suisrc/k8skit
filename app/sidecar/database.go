package sidecar

import (
	"archive/zip"
	"context"
	"crypto/sha1"
	"fmt"
	"k8skit/app"
	"k8skit/app/repo"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/suisrc/zgg/z"
	corev1 "k8s.io/api/core/v1"
)

// Inject configuration variables into the pod by database
func (patcher *Patcher) InjectConfigByDatabase(ctx context.Context, namespace string, pod *corev1.Pod) []PatchOperation {
	annotations := pod.GetAnnotations()
	if annotations == nil {
		return []PatchOperation{}
	}
	// envName
	envName, _ := pod.Annotations[patcher.InjectConfigKind]
	if envName == "" {
		return []PatchOperation{}
	}
	if len(pod.Spec.Containers) == 0 {
		return []PatchOperation{}
	}
	cidx := 0
	if idx := strings.IndexByte(envName, '#'); idx >= 0 {
		var err error
		cidx, err = strconv.Atoi(patcher.InjectConfigKind[idx+1:])
		if err != nil {
			z.Printf("Skipping configuration by database, invalid annotation [%s=%s]", patcher.InjectConfigKind, envName)
			return []PatchOperation{}
		}
		envName = envName[:idx]
	}
	kind := ""
	if idx := strings.IndexByte(envName, '.'); idx >= 0 {
		kind = envName[idx+1:]
		envName = envName[:idx]
	}
	item := &pod.Spec.Containers[cidx] // 取第一个容器作为主进程，抽取环境变量
	// version
	version := item.Image
	if idx := strings.LastIndexByte(version, ':'); idx >= 0 {
		version = version[idx+1:]
	}
	if version == "latest" {
		version = ""
	}
	// appName
	appName := item.Name
	if appName == "app" {
		appName, _ = pod.Labels["app"]
	}
	if appName == "" {
		z.Printf("Skipping configuration by database, invalid pod or container name for [%s].[%s]", pod.Name, item.Name)
		return []PatchOperation{}
	}
	appName = strings.TrimSuffix(appName, "-logtty") // 兼容 logtty
	// pathces
	patches := []PatchOperation{}
	// envConf 环境变量是必须检索的，配置文件是可选配置
	if datas := patcher.ConfxRepository.GetConfigs(envName, appName, version, "env"); len(datas) != 0 {
		for index, data := range datas {
			first := index == 0 && len(item.Env) == 0
			env := corev1.EnvVar{Name: data.Code.String, Value: data.Data.String}
			patches = append(patches, CreateArrayPatche(env, first, "/spec/containers/env"))
			item.Env = append(item.Env, env)
		}
	}
	if kind != "" && kind != "env" {
		// configuration file
		if datas := patcher.ConfxRepository.GetConfigs(envName, appName, version, kind); len(datas) > 0 {
			dirpath := patcher.InjectConfigPath
			if dirpath == "" {
				dirpath = "/confx" // 默认配置文件夹
			}
			rpath := fmt.Sprintf("/gitrepo?rand=%s&time=%d", z.GenStr("", 12), time.Now().Unix())
			for index, data := range datas {
				rpath += fmt.Sprintf("&id=%d", data.ID)
				cpath := data.Code.String
				if cpath[0] != '/' {
					cpath = filepath.Join(dirpath, cpath) // 配置文件绝对定制
				}
				spath := strings.ReplaceAll(data.Code.String, "/", "_")
				vpath := corev1.VolumeMount{Name: "confx", MountPath: cpath, SubPath: spath}
				first := index == 0 && len(item.VolumeMounts) == 0
				// volumeMount
				patches = append(patches, CreateArrayPatche(vpath, first, "/spec/containers/volumeMounts"))
				item.VolumeMounts = append(item.VolumeMounts, vpath)
			}
			{
				// 计算签名
				hstr := app.Token + rpath
				hash := sha1.New()
				hash.Write([]byte(hstr))
				sign := fmt.Sprintf("%x", hash.Sum(nil))
				rpath += fmt.Sprintf("&sign=%s", sign) // 40位长度
			}
			volume := corev1.Volume{Name: "confx", VolumeSource: corev1.VolumeSource{
				GitRepo: &corev1.GitRepoVolumeSource{
					Repository: patcher.InjectServerHost + rpath,
				},
			}}
			// volume
			patches = append(patches, CreateArrayPatche(volume, len(pod.Spec.Volumes) == 0, "/spec/volumes"))
			pod.Spec.Volumes = append(pod.Spec.Volumes, volume)
		}
	}
	return patches
}

//=========================================================================================================================

func (api *MutateApi) gitrepo(zrc *z.Ctx) {
	// 验证签名
	{
		rpath := zrc.Request.RequestURI
		sign := ""
		if idx := strings.LastIndex(rpath, "&sign="); idx >= 0 {
			sign = rpath[idx+6:]
			rpath = rpath[:idx]
		}
		if sign == "" {
			zrc.TEXT("# forbidden, sign is empty", http.StatusUnauthorized)
			return
		}
		// 计算签名
		hstr := app.Token + rpath
		hash := sha1.New()
		hash.Write([]byte(hstr))
		sig1 := fmt.Sprintf("%x", hash.Sum(nil))
		if sign != sig1 {
			zrc.TEXT("# forbidden, sign is invalid", http.StatusUnauthorized)
			return
		}
	}
	// 配置列表
	datas := []repo.ConfxDO{}
	if ids, ok := zrc.Request.URL.Query()["id"]; !ok {
		zrc.TEXT("# forbidden, id is empty", http.StatusUnauthorized)
		return
	} else {
		for _, id_ := range ids {
			id, err := strconv.ParseInt(id_, 10, 64)
			if err != nil {
				zrc.TEXT("# forbidden, id is invalid", http.StatusUnauthorized)
				return
			}
			data := api.Patcher.ConfxRepository.GetConfig1(id)
			if data == nil {
				zrc.TEXT("# forbidden, id is not found: "+id_, http.StatusUnauthorized)
				return
			}
			datas = append(datas, *data)
		}
	}
	// 处理文件
	if zrc.TraceID != "" {
		zrc.Writer.Header().Set("X-Request-Id", zrc.TraceID)
	}
	zrc.Writer.Header().Set("Content-Type", "application/zip") // 档案(archive).zip
	zrc.Writer.Header().Set("Content-Disposition", `attachment; filename="archive.zip"`)
	// zrc.Writer.WriteHeader(http.StatusOK)
	// 使用 zip 格式返回档案集合
	zw := zip.NewWriter(zrc.Writer)
	defer zw.Close()
	for _, data := range datas {
		spath := strings.ReplaceAll(data.Code.String, "/", "_")
		zf, err := zw.Create(spath)
		if err != nil {
			z.Printf("zip create error [%s]: %s", data.Code.String, err)
			continue
		}
		zf.Write([]byte(data.Data.String))
	}
}
