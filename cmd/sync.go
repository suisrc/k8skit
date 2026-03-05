package cmd

import (
	"crypto/md5"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/suisrc/zgg/z"
	"github.com/suisrc/zgg/z/zc"

	_ "github.com/go-sql-driver/mysql"
)

// go run main.go sync

var (
	C = struct {
		CmdSync SyncConfig
	}{}
)

type SyncConfig struct {
	ApiUrl string
	Token  string
}

func init() {
	z.CMD["sync"] = sync
	z.Config(&C)
	flag.StringVar(&C.CmdSync.ApiUrl, "syncapi", "", "k8s sync api url")
	flag.StringVar(&C.CmdSync.Token, "synctkn", "", "k8s sync token")
}

func sync() {
	z.Initializ()
	// parse command line arguments
	var cfs string
	flag.StringVar(&cfs, "c", "", "config file path")
	flag.Parse()
	// parse config file
	zc.LoadConfig(cfs)
	// 创建数据库 -------------------------------------------------
	// dsc, err := sqlx.ConnectDB(&zdb.C.Database, z.Println)
	// if err != nil {
	// 	fmt.Println("sync, connect db error: ", err.Error())
	// 	return
	// }
	// dsx := &sqlx.Dsx{Ex: dsc}
	// repo := sqlx.NewRepox[zdb.ConfzRepo](dsx, nil)
	// ----------------------------------------------------------
	fmt.Println("sync k8s ... -----------------")
	pageNo, pageSize := 1, 10

	infos := []map[string]any{}
	for {
		uri := fmt.Sprintf("/api/k8s/sync/v1/apps?rand=%s&time=%d&pageNo=%d&pageSize=%d&", //
			z.GenStr("", 6), time.Now().Unix(), pageNo, pageSize)
		// md5 签名
		hx5 := md5.Sum([]byte(uri + "#" + C.CmdSync.Token))
		uri = C.CmdSync.ApiUrl + uri + "sig=" + z.HexStr(hx5[:])
		z.Println("request: ", uri)
		req, err := http.Get(uri)
		if err != nil {
			fmt.Println("sync, request error: ", err.Error())
			return
		}
		rst := map[string]any{}
		if err := json.NewDecoder(req.Body).Decode(&rst); err != nil {
			fmt.Println("sync, marshal error: ", err.Error())
			return
		}
		req.Body.Close()
		if data, ok := rst["data"].([]any); !ok {
			fmt.Println("sync, data error: ", rst)
			return
		} else if len(data) == 0 {
			break
		} else {
			for _, item := range data {
				if info, ok := item.(map[string]any); !ok {
					fmt.Println("sync, item error: ", item)
					return
				} else {
					infos = append(infos, info)
				}
			}
		}
		pageNo++
		if pageNo > 1 {
			break // 测试
		}
	}

	z.Println("sync, total: ", len(infos))

}
