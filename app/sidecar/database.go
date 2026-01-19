package sidecar

import (
	"archive/tar"
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
	envName, _ := annotations[patcher.InjectConfigKind]
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
	z.Printf("Inject configuration by database, envName=[%s], appName=[%s], version=[%s]", envName, appName, version)
	// container path
	indexPath := fmt.Sprintf("/spec/containers/%d", cidx)
	// pathces
	patches := []PatchOperation{}
	// envConf 环境变量是必须检索的，配置文件是可选配置
	if datas := patcher.ConfxRepository.GetConfigs(envName, appName, version, "env"); len(datas) != 0 {
		for index, data := range datas {
			first := index == 0 && len(item.Env) == 0
			env := corev1.EnvVar{Name: data.Code.String, Value: data.Data.String}
			patches = append(patches, CreateArrayPatche(env, first, indexPath+"/env"))
			item.Env = append(item.Env, env)
		}
	}
	if kind != "" && kind != "env" {
		// configuration file
		if datas := patcher.ConfxRepository.GetConfigs(envName, appName, version, kind); len(datas) > 0 {
			dirpath, _ := annotations[patcher.InjectConfigPath]
			if dirpath == "" {
				dirpath = "/confx" // 默认配置文件夹
			}
			rpath := fmt.Sprintf("/archive?rand=%s&time=%d", z.GenStr("", 12), time.Now().Unix())
			for index, data := range datas {
				rpath += fmt.Sprintf("&id=%d", data.ID)
				cpath := data.Code.String
				if cpath[0] != '/' {
					cpath = filepath.Join(dirpath, cpath) // 配置文件绝对定制
				}
				spath := strings.ReplaceAll(data.Code.String, "/", "_")
				first := index == 0 && len(item.VolumeMounts) == 0
				// volumeMount
				vpath := corev1.VolumeMount{Name: "confx", MountPath: cpath, SubPath: spath}
				patches = append(patches, CreateArrayPatche(vpath, first, indexPath+"/volumeMounts"))
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
			tarurl := patcher.InjectServerHost + rpath
			// volume
			volume := corev1.Volume{Name: "confx", VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			}}
			patches = append(patches, CreateArrayPatche(volume, len(pod.Spec.Volumes) == 0, "/spec/volumes"))
			pod.Spec.Volumes = append(pod.Spec.Volumes, volume)
			// init container busybox:1.37.0
			// wget -q -S -O - ? | tar -x -C /conf
			initc := corev1.Container{
				Name:            "confx",
				Image:           patcher.InitArchiveImage,
				Env:             []corev1.EnvVar{{Name: "TAR_URL", Value: tarurl}},
				ImagePullPolicy: corev1.PullIfNotPresent,
				VolumeMounts:    []corev1.VolumeMount{{Name: "confx", MountPath: "/data"}},

				// Command: []string{"sh", "-c", "wget -q -O - '$(TAR_URL)' | tar -xvC /data"},
			}
			patches = append(patches, CreateArrayPatche(initc, len(pod.Spec.InitContainers) == 0, "/spec/initContainers"))
			pod.Spec.InitContainers = append(pod.Spec.InitContainers, initc)
		}
	}
	return patches
}

//=========================================================================================================================

func (api *MutateApi) archive(zrc *z.Ctx) {
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
	zrc.Writer.Header().Set("Content-Type", "application/tar") // 档案(archive).tar
	zrc.Writer.Header().Set("Content-Disposition", `attachment; filename="archive.tar"`)
	// zrc.Writer.WriteHeader(http.StatusOK)
	// 使用 zip 格式返回档案集合
	zw := tar.NewWriter(zrc.Writer)
	defer zw.Close()
	for _, data := range datas {
		spath := strings.ReplaceAll(data.Code.String, "/", "_")
		hdr := &tar.Header{Name: spath, Mode: 0666, Size: int64(len(data.Data.String))}
		err := zw.WriteHeader(hdr)
		if err != nil {
			z.Printf("tar create error [%s]: %s", data.Code.String, err)
			continue
		}
		zw.Write([]byte(data.Data.String))
	}
}
