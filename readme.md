# front2

高度集成， 封装了一个前端服务

```
front2 
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
  -dirs string
        root dir parts list (default "/zgg,/demo1/demo2")
  -eng string
        http server router engine (default "map")
  -f2show string
        show www folder resources
  -index string
        index file name (default "index.html")
  -indexs string
        index file name (default "/zgg=index.htm")
  -key string
        http server key file
  -local
        http server local mode
  -native
        use native file server
  -port int
        http server Port (default 80)
  -suff string
        replace tmpl file suffix (default ".html,.htm,.css,.map,.js")
  -tmpl string
        root router path (default "ROOT_PATH")
  -tpl string
        templates folder path
  -xrt string
        X-Request-Rt default value
```

## 使用说明

下载 main.go 文件， 创建 versio, vname 文件，然后将 nodejs 生成的 dist 文件夹中的所有内容放入 www 文件夹中。  
然后执行 go mod init ${APP} && go mod tidy && CGO_ENABLED=0 go build -ldflags "-w -s" -o ./_out/$(APP) ./ 完成构建  
将生成的单个文件，直接执行即可。