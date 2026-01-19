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
        image: suisrc/k8skit:1.3.10-kwdog2
  proxy1: |
    containers:
      - name: '[0]'
        env:
          - name: HTTP_PROXY
            value: 127.0.0.1:12012
          - name: HTTPS_PROXY
            value: 127.0.0.1:12012
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
  name: end-f1kin
spec:
  replicas: 1
  selector:
    matchLabels:
      app: end-f1kin
  template:
    metadata:
      labels:
        app: end-f1kin
        ksidecar/inject: enable
      annotations:
        ksidecar/db.config: .prop # pro.env#0, .env, .yaml, .toml ...
        # ksidecar/db.folder: /conf # config directory
        # ksidecar/configmap: default/iam#authz # namespace/configmap#attribute
    spec:
      containers:
      - name: f1kin # f1kin-logtty 优先 container.name, 如果 container.name=app, 取 labels.app 值
        image: suisrc/k8skit:1.3.10-kwlog2
        imagePullPolicy: IfNotPresent
```
使用 annotations:ksidecar/configmap 配置注入配置，多个配置用逗号隔开。  
路径 ([namespace]/)[configmap]#[attribute]  

```sql
CREATE TABLE `confx` (
  `id` int NOT NULL AUTO_INCREMENT COMMENT 'ID',
  `tag` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL COMMENT '标签: 特殊匹配， 用于追加一些特殊的匹配规则， key=val形式',
  `env` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL COMMENT '环境: DEV(开发), FAT(功能验收), UAT(用户验收), PRO(生产);\r\nDevelopment Environment;\r\nFunctional Acceptance Testing;\r\nUser Acceptance Testing;\r\nProduction Environment;',
  `app` varchar(128) DEFAULT NULL COMMENT '应用名称',
  `ver` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL COMMENT '应用版本(高版本替换低版本，聚合低版本不同的配置)',
  `kind` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL COMMENT '类型: env-ref/yaml-ref(引用), env(环境), json，prop, yaml, toml ...',
  `code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL COMMENT '编码: 应用名称_版本/环境名称/文件名称（''/''开头abspath)',
  `data` mediumtext CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci COMMENT '内容',
  `dkey` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL COMMENT '秘钥，与应用秘钥合用，完成data加密',
  `disable` int DEFAULT '0' COMMENT '禁用(禁用后，低版本会被删除)',
  `deleted` int DEFAULT '0' COMMENT '删除(删除后，使用低版本替代)',
  `updated` datetime DEFAULT NULL COMMENT '更新时间',
  `updater` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL COMMENT '更新者',
  `created` datetime DEFAULT NULL COMMENT '创建时间',
  `creater` varchar(255) DEFAULT NULL COMMENT '创建者',
  `version` int DEFAULT '0' COMMENT '版本',
  PRIMARY KEY (`id`),
  KEY `conf_app` (`app`),
  KEY `conf_code` (`code`),
  KEY `conf_env` (`env`),
  KEY `conf_ver` (`ver` DESC) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
```

[ksidecar/db.config] 通过数据库 confx 表注入相关配置  
[ksidecar/db.folder] 如果注入的是配置文件，则指定配置文件所在目录  

## 鸣谢
[ExpediaGroup](https://github.com/ExpediaGroup/kubernetes-sidecar-injector)

