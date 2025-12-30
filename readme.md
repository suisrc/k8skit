# k8skit

高度集成， 封装了一个前端服务

```
k8skit 
  -addr string
        http server addr (default "0.0.0.0")
  -api string
        http server api path
  -c string
        config file path
  -crt string
        http server cer file
  -debug
        debug mode
  -eng string
        http server router engine (default "map")
  -key string
        http server key file
  -local
        http server local mode
  -port int
        http server Port (default 80)
  -tpl string
        templates folder path
  -xrt string
        X-Request-Rt default value
```

## 使用说明


 k8s集群工具箱

[zgg](https://github.com/suisrc/zgg.git) Web服务框架

[k8skit](https://github.com/suisrc/k8skit.git) k8s工具包
- [kubesider](https://github.com/suisrc/k8skit/tree/sidecar): k8s 边车注入服务
- [front2](https://github.com/suisrc/k8skit/tree/front2): 前端部署服务， 取代nginx部署前段
- [kwdog2](https://github.com/suisrc/k8skit/tree/kwdog2): k8s 容器日志、监控、鉴权服务
- [fluent](https://github.com/suisrc/k8skit/tree/fluent): fluentd 日志HTTP接受服务