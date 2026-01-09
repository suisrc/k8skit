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

工具包内容：  

[k8skit](https://github.com/suisrc/k8skit.git) k8s工具包
- [sidecar](https://github.com/suisrc/k8skit/tree/sidecar): kube-injector + fake-ssl, 边车注入 + 模拟SSL
- [fluentbit](https://github.com/suisrc/k8skit/tree/fluentbit): 支持阿里云 sls 日志服务
- [front2](https://hub.docker.com/r/suisrc/k8skit/tags?name=front2): 前端部署服务， 取代nginx部署前段
- [kwdog2](https://hub.docker.com/r/suisrc/k8skit/tags?name=kwdog2): k8s 容器日志、监控、鉴权服务， kwdog2 + proxy2
- [kwlog2](https://hub.docker.com/r/suisrc/k8skit/tags?name=kwlog2): fluentbit 日志收集服务，之后上报到 http 服务器