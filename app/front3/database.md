
```sql
CREATE TABLE `confa` (
  `id` int NOT NULL AUTO_INCREMENT,
  `akey` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL COMMENT '访问KEY',
  `secret` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL COMMENT 'KEY秘钥',
  `role` varchar(255) DEFAULT NULL COMMENT '角色',
  `remarks` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL COMMENT '备注信息',
  `disable` int DEFAULT '0' COMMENT '禁用标记',
  `deleted` int DEFAULT '0' COMMENT '删除标记',
  `created` datetime DEFAULT NULL COMMENT '创建时间',
  `creater` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL COMMENT '创建者',
  `version` int DEFAULT '0' COMMENT '版本',
  PRIMARY KEY (`id`),
  KEY `conf_akey` (`akey`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
```

```sql
CREATE TABLE `fronta` (
  `id` int NOT NULL AUTO_INCREMENT,
  `tag` varchar(255) DEFAULT NULL COMMENT '标签',
  `name` varchar(255) DEFAULT NULL COMMENT '名称',
  `app` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL COMMENT '应用',
  `ver` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL COMMENT '版本，不为空指定版本',
  `domain` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL COMMENT '域名',
  `rootdir` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL COMMENT '根目录',
  `priority` varchar(255) DEFAULT NULL COMMENT '优先级',
  `routers` text COMMENT '路由配置',
  `disable` int DEFAULT '0' COMMENT '禁用标识，最高版本禁用，所有版本不可使用',
  `deleted` int DEFAULT '0' COMMENT '删除标识，只对当前版本，其他版本依然可用',
  `updated` datetime DEFAULT NULL COMMENT '更新时间',
  `updater` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL COMMENT '更新者',
  `created` datetime DEFAULT NULL COMMENT '创建时间',
  `creater` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL COMMENT '创建者',
  `version` int DEFAULT '0' COMMENT '版本',
  PRIMARY KEY (`id`),
  KEY `f2_domain` (`domain`),
  KEY `f2_app` (`app`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
```

```sql
CREATE TABLE `frontv` (
  `id` int NOT NULL AUTO_INCREMENT,
  `tag` varchar(255) DEFAULT NULL COMMENT '标签',
  `aid` int NOT NULL COMMENT '应用',
  `ver` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL COMMENT '版本， 可以与 image中的版本不一致，比如增加小版本',
  `image` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL COMMENT '前端镜像',
  `tproot` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL COMMENT '替换根目录标识 \r\n要部署CDN情况，该系必填\r\nnone 或 @~ 或 /ROOT_PATH(默认)...\r\n解决打包 rootdir 和部署 rootdir 不一致',
  `indexpath` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL COMMENT '索引文件, 默认 index.html',
  `indexs` text COMMENT '索引列表， /tas=,/tas/embed=index.htm',
  `imagepath` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL COMMENT '前端文件(在镜像中的)， 默认 /opt/www',
  `recache` int DEFAULT '0' COMMENT '是否重新缓存',
  `cdnname` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL COMMENT 'CDN加速域名， 不为空表示CDN可用，cdnuse控制',
  `cdnpath` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL COMMENT '使用CDN所在文件夹， 不包含app和ver',
  `cdnuse` int DEFAULT '0' COMMENT '是否使用cdn',
  `cdnrew` int DEFAULT '1' COMMENT '标记重新cdn，一次有效',
  `started` datetime DEFAULT NULL COMMENT '发布时间，生效时间， 可用于提前发布',
  `indexhtml` mediumtext CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci COMMENT '索引页面，人工指定, 优先级最高，存在即直接返回',
  `disable` int DEFAULT NULL COMMENT '禁用标识',
  `deleted` int DEFAULT NULL COMMENT '删除标识',
  `updated` datetime DEFAULT NULL COMMENT '更新时间',
  `updater` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL COMMENT '更新者',
  `created` datetime DEFAULT NULL COMMENT '创建时间',
  `creater` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL COMMENT '创建者',
  `version` int DEFAULT '0' COMMENT '版本',
  PRIMARY KEY (`id`),
  KEY `f2_ver` (`ver`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
```