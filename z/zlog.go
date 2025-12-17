package z

import (
	"log"
	// "k8s.io/klog/v2"
)

var (
	Println = log.Println
	Printf  = log.Printf
	Fatal   = log.Fatal
	Fatalf  = log.Fatalf

	// Println = klog.Infoln
	// Printf  = klog.Infof
	// Fatal   = klog.Fatal
	// Fatalf  = klog.Fatalf
)
