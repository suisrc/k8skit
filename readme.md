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
- [sidecar](https://github.com/suisrc/k8skit/tree/sidecar): k8s 边车注入服务， 编排监控备份
- [front3](https://github.com/suisrc/k8skit/tree/front3): 扩展前端部署服务，提供将前端部署到S3CDN的服务
- [fluentbit](https://github.com/suisrc/k8skit/tree/fluentbit): fluentbit 日志服务扩展
- [fmesui](https://github.com/suisrc/k8skit/tree/fmesui): FMES平台集群控制UI

## 其他模块

- [alidns-webhook](https://github.com/suisrc/alidns-webhook.git) 为 cert-manager 模块提供 阿里云, 腾讯云， 华为云 DNS 支持