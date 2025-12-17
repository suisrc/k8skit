package main

import (
	_ "kube-sidecar/app"
	_ "kube-sidecar/cmd"
	"kube-sidecar/z"
)

/**
 * 程序入口
 */
func main() {
	z.Execute()
}
