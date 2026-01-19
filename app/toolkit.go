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
)

var (
	Token string

	// C = struct {
	// }{}
)

func init() {
	z.Register("11-app.init", func(zgg *z.Zgg) z.Closed {
		// 创建 k8sclient
		// cli, err := CreateClient(z.C.Server.Local)
		cli, err := CreateClient(true)
		if err != nil {
			zgg.ServeStop("create k8s client error: ", err.Error()) // 初始化失败，直接退出
			return nil
		}
		z.Println("[_k8scli_]: create k8s client success: local=", z.C.Server.Local)
		zgg.SvcKit.Set("k8sclient", cli) // 注册 k8sclient
		return nil
	})
}

// -----------------------------------------------------------------------

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
		z.Println("[_k8scli_]: using local kubeconfig.")
		kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	z.Println("[_k8scli_]: using in cluster kubeconfig.")
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
