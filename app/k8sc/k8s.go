package k8sc

import (
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/pkg/errors"
	"github.com/suisrc/zgg/z"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	Token string
)

func init() {
	z.Register("11-k8s.init", func(zgg *z.Zgg) z.Closed {
		// 创建 k8sclient
		cli, err := CreateClient(z.C.Server.Local)
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

func TrimYamlString(str string) string {
	sbr := strings.Builder{}
	for line := range strings.SplitSeq(str, "\n") {
		sbr.WriteString(strings.TrimRightFunc(line, unicode.IsSpace))
		sbr.WriteRune('\n')
	}
	return strings.TrimRightFunc(sbr.String(), unicode.IsSpace)
}

// -----------------------------------------------------------------------
