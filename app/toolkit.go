package app

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/suisrc/zgg/z"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

var (
	namespace_ = ""

	C = struct {
		Token            string
		InjectAnnotation string
		InjectDefaultKey string
	}{}
)

func init() {
	z.Register("11-app.init", func(zgg *z.Zgg) z.Closed {
		// 创建 k8sclient
		client, err := CreateClient(z.C.Server.Local)
		if err != nil {
			klog.Error("create k8s client error: ", err.Error())
			zgg.ServeStop() // 初始化失败，直接退出
			return nil
		}
		// z.RegSvc(srv.GetSvcKit(), client)
		klog.Info("create k8s client success: local=", z.C.Server.Local)
		zgg.SvcKit.Set("k8sclient", client) // 注册 k8sclient
		return nil
	})
}

// -----------------------------------------------------------------------

// 获取当前命名空间 k8s namespace
func K8sNS() string {
	if namespace_ != "" {
		return namespace_
	}
	ns, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		z.Printf("unable to read namespace: %s, return 'default'", err.Error())
		namespace_ = "default"
	} else {
		namespace_ = string(ns)
	}
	return namespace_
}

// CreateClient Create the server
func CreateClient(local bool) (*kubernetes.Clientset, error) {
	cfg, err := BuildConfig(local)
	if err != nil {
		return nil, errors.Wrapf(err, "error setting up cluster config")
	}
	return kubernetes.NewForConfig(cfg)
}

// BuildConfig Build the config
func BuildConfig(local bool) (*rest.Config, error) {
	if local {
		klog.Info("using local kubeconfig.")
		kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	klog.Info("using in cluster kubeconfig.")
	return rest.InClusterConfig()
}

// -----------------------------------------------------------------------

func PostJson(req *http.Request) error {
	if req.Method != http.MethodPost {
		return fmt.Errorf("wrong http verb. got %s", req.Method)
	}
	if req.Body == nil {
		return errors.New("empty body")
	}
	contentType := req.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		return fmt.Errorf("wrong content type. expected 'application/json', got: '%s'", contentType)
	}
	return nil
}
