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
	"github.com/suisrc/zgg/z/ze/sqlx"

	"k8skit/app/zdb"

	_ "github.com/go-sql-driver/mysql"
)

// go run main.go sync

var (
	C = struct {
		CmdSync SyncConfig
	}{}
)

type SyncConfig struct {
	ApiUrl  string
	Token   string
	Version int64
	User    string
}

func init() {
	z.CMD["sync"] = sync
	z.Config(&C)
	flag.StringVar(&C.CmdSync.ApiUrl, "syncapi", "", "k8s sync api url")
	flag.StringVar(&C.CmdSync.Token, "synctkn", "", "k8s sync token")
	flag.Int64Var(&C.CmdSync.Version, "syncver", 1, "k8s sync version")
	flag.StringVar(&C.CmdSync.User, "syncusr", "syncuser", "k8s sync user")
}

func sync() {
	z.Initializ()
	// parse command line arguments
	var cfs string
	flag.StringVar(&cfs, "c", "", "config file path")
	flag.Parse()
	// parse config file
	zc.LoadConfig(cfs)
	// ----------------------------------------------------------
	// cli, err := k8sc.CreateClient(z.C.Server.Local)
	// if err != nil {
	// 	fmt.Println("create k8s client error: ", err.Error()) // 初始化失败，直接退出
	// 	return
	// }
	dsc, err := sqlx.ConnectDB(&zdb.C.Database, z.Println)
	if err != nil {
		fmt.Println("sync, connect db error: ", err.Error())
		return
	}
	dsx := &sqlx.Dsx{Ex: dsc}
	zck8sRepo := sqlx.NewRepox[zdb.Zck8sRepo](dsx, nil)
	// zconfRepo := sqlx.NewRepox[zdb.ZconfRepo](dsx, nil)
	// 删除原有数据
	zck8sRepo.DeleteBy(nil, fmt.Sprintf("version=%d", C.CmdSync.Version))
	// zconfRepo.DeleteBy(nil, fmt.Sprintf("version=%d", C.CmdSync.Version))
	// ----------------------------------------------------------
	if err := sync_(zck8sRepo, "apps"); err != nil {
		z.Println(err.Error())
		return
	}
	if err := sync_(zck8sRepo, "ings"); err != nil {
		z.Println(err.Error())
		return
	}
	z.Println("sync, finally")

}

func sync_(zck8sRepo *zdb.Zck8sRepo, key string) error {
	fmt.Printf("sync k8s %s ... -----------------", key)
	pageNo, pageSize := 1, 10
	for {
		uri := fmt.Sprintf("/api/k8s/sync/v1/%s?rand=%s&time=%d&pageNo=%d&pageSize=%d&", //
			key, z.GenStr("", 6), time.Now().Unix(), pageNo, pageSize)
		// md5 签名
		hx5 := md5.Sum([]byte(uri + "#" + C.CmdSync.Token))
		uri = C.CmdSync.ApiUrl + uri + "sig=" + z.HexStr(hx5[:])
		z.Println("request: ", uri)
		req, err := http.Get(uri)
		if err != nil {
			return fmt.Errorf("sync, request error: %s", err.Error())
		}
		rst := map[string]any{}
		if err := json.NewDecoder(req.Body).Decode(&rst); err != nil {

			return fmt.Errorf("sync, marshal error: %s", err.Error())
		}
		req.Body.Close()
		if data, ok := rst["data"].([]any); !ok {
			return fmt.Errorf("sync, data error: %s", zc.ToStr(rst))
		} else if len(data) == 0 {
			break // 已经没有数据了
		} else {
			for _, item := range data {
				info, ok := item.(map[string]any)
				if !ok {
					return fmt.Errorf("sync, item error: %s", zc.ToStr(item))
				}
				ck8s := zdb.Zck8sDO{}
				if val, ok := info["json"]; ok {
					bts, _ := json.MarshalIndent(val, "", "  ")
					ck8s.Json = sqlx.NewString(string(bts))
					delete(info, "json")
				}
				if val, ok := info["yaml"]; ok {
					str, _ := val.(string)
					ck8s.Yaml = sqlx.NewString(str)
					delete(info, "yaml")
				}
				if val, ok := info["label"].(string); ok {
					ck8s.Label = sqlx.NewString(val)
				}
				if val, ok := info["kind"].(string); ok {
					ck8s.Kind = sqlx.NewString(val)
				}
				if val, ok := info["name"].(string); ok {
					ck8s.Name = sqlx.NewString(val)
				}
				if val, ok := info["namespace"].(string); ok {
					ck8s.Namespace = sqlx.NewString(val)
				}
				bts, err := json.MarshalIndent(info, "", "  ")
				if err != nil {
					return fmt.Errorf("sync, json marshal error: %s", err.Error())
				}
				ck8s.Data = sqlx.NewString(string(bts))
				ck8s.Created = sqlx.NewTime(time.Now())
				ck8s.Creater = sqlx.NewString(C.CmdSync.User)
				ck8s.Version = sqlx.NewInt64(C.CmdSync.Version)

				zck8sRepo.Insert(nil, &ck8s)
				z.Printf("sync, insert %s: %s", key, ck8s.Name.String)
			}
		}
		pageNo++
		// if pageNo > 1 {
		// 	break // 测试
		// }
	}
	return nil
}
