package front3

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/suisrc/zgg/z"
)

// 提供webhook，对影响进行升级

func (aa *F3Serve) WebHook(zrc *z.Ctx) {
	switch zrc.Request.URL.Query().Get("method") {
	case "update.image":
		aa.updateVersion(zrc)
	default:
		zrc.TEXT("no", http.StatusOK)
	}
}

func (aa *F3Serve) updateVersion(zrc *z.Ctx) {
	qry := zrc.Request.URL.Query()
	key := qry.Get("akey")
	if key == "" {
		zrc.TEXT("token is empty", http.StatusOK)
		return
	}
	akey, err := aa.KeyRepo.GetByAkey(key)
	if err != nil || akey == nil {
		zrc.TEXT("token is invalid", http.StatusOK)
		return
	} else if akey.Disable {
		zrc.TEXT("token is disable", http.StatusOK)
		return
	} else if role := akey.Role.String; role == "" {
		zrc.TEXT("role is empty", http.StatusOK)
		return
	} else if strings.HasSuffix(role, ".*") && //
		strings.HasPrefix("front3.update.image", role[0:len(role)-1]) {
		// pass
	} else if role == "front3.update.image" {
		// pass
	} else {
		zrc.TEXT("role is forbidden", http.StatusOK)
		return
	}
	image := qry.Get("image")
	skey := strings.SplitN(image, ":", 2)
	if len(skey) != 2 {
		zrc.TEXT("image is invalid", http.StatusOK)
		return
	}
	if vers, err := aa.VerRepo.GetByImage(image); err == nil && len(vers) > 0 {
		zrc.TEXT("image is exist", http.StatusOK)
		return
	}
	ikey, iver := skey[0], skey[1]
	if app := qry.Get("app"); app != "" {
		// 通过应用编码查找应用
		appInfo, err := aa.AppRepo.GetByApp(app)
		if err != nil {
			zrc.TEXT("application not found: "+app, http.StatusOK)
			return
		}
		vpp := appInfo.GVP()
		verInfo, err := aa.VerRepo.GetTop1ByVppAndVer(vpp, "") // 最新版
		if err != nil {
			zrc.TEXT("application version not found: "+app, http.StatusOK)
			return
		}
		verInfo.ID = 0
		verInfo.CdnName.Valid = false
		verInfo.CdnPath.Valid = false
		verInfo.CdnUse.Valid = false
		verInfo.CdnRew.Valid = false
		verInfo.Image = sql.NullString{String: image, Valid: true}
		verInfo.Vpp = vpp
		verInfo.Ver = iver
		verInfo.Started = sql.NullTime{Time: time.Now(), Valid: true}
		aa.VerRepo.Insert(verInfo)
	} else {
		// 使用镜像 ikey 查询和替换应用版本， 需要考虑存在多个的情况
		vers, err := aa.VerRepo.GetByImageName(ikey)
		if err != nil || len(vers) == 0 {
			zrc.TEXT("application version not found: "+ikey, http.StatusOK)
			return
		}
		// 遍历更新应用版本
		for _, ver := range vers {
			ver.ID = 0
			ver.CdnName.Valid = false
			ver.CdnPath.Valid = false
			ver.CdnUse.Valid = false
			ver.CdnRew.Valid = false
			ver.Image = sql.NullString{String: image, Valid: true}
			ver.Ver = iver // 只更新版本和镜像
			ver.Started = sql.NullTime{Time: time.Now(), Valid: true}
			aa.VerRepo.Insert(&ver)
		}
	}

	zrc.TEXT("ok", http.StatusOK)
}
