module kube-sidecar

go 1.25.5

// replace github.com/suisrc/zgg => ../zgg

require (
	github.com/suisrc/zgg v0.0.6
	k8s.io/klog/v2 v2.130.1
)

require github.com/go-logr/logr v1.4.1 // indirect
