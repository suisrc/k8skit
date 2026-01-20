package sidecar

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/samber/lo"
	"github.com/suisrc/zgg/z"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Sidecar Kubernetes Sidecar Injector schema
type Sidecar struct {
	InitContainers   []corev1.Container            `yaml:"initContainers"`
	Containers       []corev1.Container            `yaml:"containers"`
	Volumes          []corev1.Volume               `yaml:"volumes"`
	ImagePullSecrets []corev1.LocalObjectReference `yaml:"imagePullSecrets"`
	Annotations      map[string]string             `yaml:"annotations"`
	Labels           map[string]string             `yaml:"labels"`
}

// ConfigmapSidecarData
func (patcher *Patcher) ConfigmapSidecarData(ctx context.Context, namespace, configmapSidecarName string, pod corev1.Pod) (*Sidecar, error) {
	configKey := patcher.Config.DefaultKey // key of the configmap
	configName := configmapSidecarName
	if idx := strings.IndexRune(configName, '/'); idx > 0 {
		namespace = configName[:idx]
		configName = configName[idx+1:]
	}
	if idx := strings.IndexRune(configName, '#'); idx > 0 {
		configKey = configName[idx+1:]
		configName = configName[:idx]
	}
	if configKey == "" {
		return nil, fmt.Errorf("invalid configmap name: %s.%s.[empty]", namespace, configName)
	}
	configMap, err := patcher.K8sClient.CoreV1().ConfigMaps(namespace).Get(ctx, configName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if sidecarStr, ok := configMap.Data[configKey]; !ok {
		return nil, fmt.Errorf("configmap %v does not contain key %v", configName, configKey)
	} else {
		sidecar := &Sidecar{}
		if err := yaml.Unmarshal([]byte(sidecarStr), sidecar); err != nil {
			return nil, fmt.Errorf("failed to parse configmap %v key %v: %v", configName, configKey, err)
		}
		return sidecar, nil
	}
}

// ConfigmapSidecarNames
func (patcher *Patcher) ConfigmapSidecarNames(namespace string, pod corev1.Pod) []string {
	podName := pod.GetName()
	if podName == "" {
		podName = pod.GetGenerateName()
	}
	annotations := map[string]string{}
	if pod.GetAnnotations() != nil {
		annotations = pod.GetAnnotations()
	}
	if sidecars, ok := annotations[patcher.Config.Annotation]; ok {
		parts := lo.Map(strings.Split(sidecars, ","), func(part string, _ int) string { // Map[string, string]
			return strings.TrimSpace(part)
		})

		if len(parts) > 0 {
			z.Printf("sidecar injection for %v/%v: sidecars: %v", namespace, podName, sidecars)
			return parts
		}
	}
	z.Printf("Skipping mutation for [%v]. No action required", podName)
	return nil
}

//=========================================================================================================================

// func (patcher *Patcher) FixSidecarByPodAnnotations(sidecar *Sidecar, annotations map[string]string) {
// 	if (annotations == nil) || (len(annotations) == 0) {
// 		return
// 	}
// 	envPrefixKey := patcher.InjectPrefix + "/env."
// 	for envName, envValue := range annotations {
// 		if strings.HasPrefix(envName, envPrefixKey) {
// 			envName = envName[len(envPrefixKey):]
// 			for idx := range sidecar.InitContainers {
// 				patcher.FixSidecarContainerEnvValue(&sidecar.Containers[idx], envName, envValue)
// 			}
// 			for idx := range sidecar.Containers {
// 				patcher.FixSidecarContainerEnvValue(&sidecar.Containers[idx], envName, envValue)
// 			}
// 		}
// 	}
// }
//
// func (patcher *Patcher) FixSidecarContainerEnvValue(container *corev1.Container, envName, envValue string) {
// 	edx := -1
// 	for jdx := range container.Env {
// 		if container.Env[jdx].Name == envName {
// 			edx = jdx
// 			break
// 		}
// 	}
// 	if edx >= 0 {
// 		// update existing env value
// 		container.Env[edx].Value = envValue
// 	} else if container.Env == nil {
// 		container.Env = []corev1.EnvVar{{Name: envName, Value: envValue}}
// 	} else {
// 		// add new env value
// 		container.Env = append(container.Env, corev1.EnvVar{Name: envName, Value: envValue})
// 	}
// }

//=========================================================================================================================

// CreateContainersPatches
func CreateContainersPatches(newContainers []corev1.Container, existingContainers *[]corev1.Container, path string) []PatchOperation {
	if len(newContainers) == 0 {
		// no new containers, so no patch necessary
		return []PatchOperation{}
	}
	regexMatchName, _ := regexp.Compile(`^[\w-\.]+$`)
	if len(*existingContainers) == 0 {
		// pass merge patch if no existing containers
		filterContainers := []corev1.Container{}
		for _, item := range newContainers {
			if ok := regexMatchName.MatchString(item.Name); ok {
				filterContainers = append(filterContainers, item)
			}
		}
		return CreateArrayPatches(filterContainers, existingContainers, path)
	}
	// merge existing and new containers
	var patches []PatchOperation
	regexMatchIdx, _ := regexp.Compile(`^\[\d+\]$`)
	for index, item := range newContainers {
		if !regexMatchIdx.MatchString(item.Name) {
			idx := FindIndexArrayObject(existingContainers, func(that corev1.Container) bool {
				if item.Name == that.Name {
					// found matching container name
					return true
				} else if strings.HasPrefix(item.Name, "[") && strings.HasSuffix(item.Name, "]") {
					// check for name match with regex, [regex_str] => match by regex
					if ok, _ := regexp.MatchString(item.Name[1:len(item.Name)-1], that.Name); ok {
						return true
					}
				}
				// no match
				return false
			})
			if idx < 0 {
				if ok := regexMatchName.MatchString(item.Name); ok {
					// no existing container with same name, so add it
					first := index == 0 && len(*existingContainers) == 0
					patches = append(patches, CreateArrayPatche(item, first, path))
					*existingContainers = append(*existingContainers, item)
				}
				// else ignore
				continue
			}
			// existing container with same name, so merge it
			item.Name = fmt.Sprintf("[%d]", idx)
		}
		idx, _ := strconv.Atoi(item.Name[1 : len(item.Name)-1]) // [0] -> current running container
		if idx < 0 || idx >= len(*existingContainers) {
			continue // skip non-exist pod container, not found in existingContainers
		}
		// update existing pod container
		existContainer := &(*existingContainers)[idx]
		indexPath := fmt.Sprintf("%s/%d", path, idx)

		// init container fileds, nil is not allowed
		if existContainer.EnvFrom == nil {
			existContainer.EnvFrom = []corev1.EnvFromSource{}
		}
		if existContainer.Env == nil {
			existContainer.Env = []corev1.EnvVar{}
		}
		if existContainer.VolumeMounts == nil {
			existContainer.VolumeMounts = []corev1.VolumeMount{}
		}
		if existContainer.VolumeDevices == nil {
			existContainer.VolumeDevices = []corev1.VolumeDevice{}
		}
		// update existing pod container
		patches = append(patches, CreateArrayPatches(item.EnvFrom, &existContainer.EnvFrom, indexPath+"/envFrom")...)
		patches = append(patches, CreateArrayPatches(item.Env, &existContainer.Env, indexPath+"/env")...)
		patches = append(patches, CreateArrayPatches(item.VolumeMounts, &existContainer.VolumeMounts, indexPath+"/volumeMounts")...)
		patches = append(patches, CreateArrayPatches(item.VolumeDevices, &existContainer.VolumeDevices, indexPath+"/volumeDevices")...)

		if item.LivenessProbe != nil && existContainer.LivenessProbe == nil {
			patches = append(patches, PatchOperation{Op: "add", Path: indexPath + "/livenessProbe", Value: item.LivenessProbe})
			existContainer.LivenessProbe = item.LivenessProbe
		}
		if item.ReadinessProbe != nil && existContainer.ReadinessProbe == nil {
			patches = append(patches, PatchOperation{Op: "add", Path: indexPath + "/readinessProbe", Value: item.ReadinessProbe})
			existContainer.ReadinessProbe = item.ReadinessProbe
		}
		if item.Lifecycle != nil && existContainer.Lifecycle == nil {
			patches = append(patches, PatchOperation{Op: "add", Path: indexPath + "/lifecycle", Value: item.Lifecycle})
			existContainer.Lifecycle = item.Lifecycle
		}
		if item.SecurityContext != nil && existContainer.SecurityContext == nil {
			patches = append(patches, PatchOperation{Op: "add", Path: indexPath + "/securityContext", Value: item.SecurityContext})
			existContainer.SecurityContext = item.SecurityContext
		}
	}
	return patches
}

// CreateArrayPatches
func CreateArrayPatches[T any](newCollection []T, existingCollection *[]T, path string /*, exist func(T, T) bool*/) []PatchOperation {
	if len(newCollection) == 0 {
		// no new collection, so no patch necessary
		return []PatchOperation{}
	}
	var patches []PatchOperation
	for index, item := range newCollection {
		first := index == 0 && len(*existingCollection) == 0
		// if !first && exist != nil {
		// 	idx := FindIndexArrayObject(existingCollection, func(that T) bool {
		// 		return exist(item, that)
		// 	})
		// 	// item is exist in existingCollection，pass merge patch
		// 	if idx >= 0 {
		// 		continue
		// 	}
		// }
		patches = append(patches, CreateArrayPatche(item, first, path))
		*existingCollection = append(*existingCollection, item)
	}
	return patches
}

// CreateArrayPatche
func CreateArrayPatche[T any](item T, first bool, path string) PatchOperation {
	var value any
	if !first {
		path = path + "/-"
		value = item
	} else {
		value = []T{item}
	}
	return PatchOperation{
		Op:    "add",
		Path:  path,
		Value: value,
	}
}

// CreateObjectPatches
func CreateObjectPatches(newMap map[string]string, existingMap *map[string]string, path string, override bool) []PatchOperation {
	if len(newMap) == 0 {
		// no new map, so no patch necessary
		return []PatchOperation{}
	}
	var patches []PatchOperation
	if existingMap == nil {
		patches = append(patches, PatchOperation{
			Op:    "add",
			Path:  path,
			Value: newMap,
		})
	} else {
		for key, value := range newMap {
			if _, ok := (*existingMap)[key]; !ok || (ok && override) {
				key = escapeJSONPath(key)
				op := "add"
				if ok {
					op = "replace"
				}
				patches = append(patches, PatchOperation{
					Op:    op,
					Path:  path + "/" + key,
					Value: value,
				})
				(*existingMap)[key] = value
			}
		}
	}
	return patches
}

// FindIndexArrayObject
func FindIndexArrayObject[T any](items *[]T, exist func(T) bool) int {
	for idx, item := range *items {
		if exist(item) {
			return idx
		}
	}
	return -1
}

// Escape keys that may contain `/`s or `~`s to have a valid patch
// Order matters here, otherwise `/` --> ~01, instead of ~1
func escapeJSONPath(k string) string {
	k = strings.ReplaceAll(k, "~", "~0")
	return strings.ReplaceAll(k, "/", "~1")
}
