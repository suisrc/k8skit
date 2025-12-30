![helm-release-gha](https://github.com/suisrc/k8skit/actions/workflows/ci.yml/badge.svg?branch=sidecar)
[![License](https://img.shields.io/badge/license-Apache%20License%202.0-blue.svg)](https://github.com/suisrc/k8skit/blob/main/LICENSE)

# Sidecar

Kubernetes Mutating Webhook

边车注入服务是使用kubernates中MutatingWebhookConfiguration，可对进入k8s的编排进行修改，简化k8s的编。  
注意它只是辅助功能，没有它也可以，只不过配置更复杂，关于权限的编排没有办法统一管理而已。  

1. 边车注入服务
2. 虚拟证书签发服务


## Using this webhook

### 定义注入配置

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: dev-kwdog
  namespace: dev-env
data:
  kwdog1: |
    containers:
      - name: kwdog
        image: suisrc/openresty:1.21.4.1-az-32
  proxyp: | 
    containers:
      - name: kwdog
        env:
          - name: KS_PROXYDOG
            value: pxy_p
  proxyh: |
    containers:
      - name: '[0]'
        env:
          - name: HTTP_PROXY
            value: 127.0.0.1:12012
          - name: HTTPS_PROXY
            value: 127.0.0.1:12012
      - name: kwdog
        env:
          - name: KS_PROXYDOG
            value: pxy_p,pxy_h
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: tst-kwdog
  namespace: tst-env
data:
  kwdog1: |
    containers:
      - name: kwdog
        image: suisrc/openresty:1.21.4.1-az-32
  proxyp: | 
    containers:
      - name: kwdog
        env:
          - name: KS_PROXYDOG
            value: pxy_p
  proxyh: |
    containers:
      - name: '[0]'
        env:
          - name: HTTP_PROXY
            value: 127.0.0.1:12012
          - name: HTTPS_PROXY
            value: 127.0.0.1:12012
      - name: kwdog
        env:
          - name: KS_PROXYDOG
            value: pxy_p,pxy_h
```

1. 注入配置定义在ConfigMap中，可以在任意空间中定义。
2. 定义配置会与提交的编排进行合并。
3. 合并原则是 [name] 相同。
4. [0], 表示第一个容器， [1], 表示第二个容器。

### 标记注入配置

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tst-app
spec:
  replicas: 1
  selector:
    matchLabels:
      app: tst-app
  template:
    metadata:
      labels:
        app: tst-app
        ksidecar/inject: enable
      annotations:
        ksidecar/configmap: >-
          tst-kwdog#authx,
          tst-kwdog#iam,
          tst-kwdog#proxya,
          kube-ksidecar/ca-tools#getter.dev,
          kube-ksidecar/ca-tools#java11

```
使用 annotations:ksidecar/configmap 配置注入配置，多个配置用逗号隔开。  
路径 [namespace]/[configmap]#[configmap-key]  

## 鸣谢
[ExpediaGroup](https://github.com/ExpediaGroup/kubernetes-sidecar-injector)

