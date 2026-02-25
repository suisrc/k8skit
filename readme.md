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


[zgg](https://github.com/suisrc/zgg.git) Web服务框架

[k8skit](https://github.com/suisrc/k8skit.git) k8s工具包
- [sidecar](https://github.com/suisrc/k8skit/tree/sidecar): k8s 边车注入服务
- [wgetar](https://hub.docker.com/r/suisrc/k8skit/tags?name=wgetar): k8s 边车注入服务中，对于配置文件的获取服务， 基于busybox 的 wget+tar
- [front2](https://hub.docker.com/r/suisrc/k8skit/tags?name=front2): 前端部署服务， 取代 nginx 作为前端容器，提供灵活的根路径配置等
- [kwdog2](https://hub.docker.com/r/suisrc/k8skit/tags?name=kwdog2): 由 kwdog2 + proxy2 组成的服务， 提供了 k8s 容器日志、监控、鉴权服务
- [kwlog2](https://hub.docker.com/r/suisrc/k8skit/tags?name=kwlog2): fluentbit 日志HTTP接受服务, 提供简单的日志存储和查询服务
- [front3](https://github.com/suisrc/k8skit/tree/front3): 扩展前端部署服务，提供将前端部署到S3CDN的服务

## 其他模块

- [alidns-webhook](https://github.com/suisrc/alidns-webhook.git) 为 cert-manager 模块提供 阿里云, 腾讯云， 华为云 DNS 支持
- [fmesui](https://github.com/suisrc/k8skit/tree/fmesui): FMES平台集群控制UI