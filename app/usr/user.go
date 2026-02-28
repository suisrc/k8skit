package usr

type UserInfo struct {
	Name     string `json:"name"` // 不为空为已经登录
	Nickname string `json:"password"`
}

type User struct {
	UserInfo

	Session  string `json:"session"` // 令牌
	IsLogin  bool   `json:"islogin"` // 状态
	Nonces   string `json:"nonces"`
	ExpireAt int64  `json:"expireat"`
}

// {
//     "success": true,
//     "data": {
//         "app_name": "",
//         "authc_status": true,
//         "avatar": "https://fmes.oss-cn-shanghai.aliyuncs.com/s3/18981/aRA000000032pofM0LLypvuN.jpg?Expires=1770881798&OSSAccessKeyId=LTAI4GH389TyyoCe1VXKeAEB&Signature=vgdTVACMtXg2T855pAC4Axn5LKQ%3D&x-oss-process=image/quality%2Cq_85/resize%2Ch_100",
//         "copyright_code": "皖ICP备2025101709号-1",
//         "copyright_text": "Copyright © 2025 masjingyao.top",
//         "footer_links": "[]",
//         "logo_icon": "https://cdn.masjingyao.top/jingyao_logo.png",
//         "msg_all_uri": "https://account.masjingyao.top/notification/allMessage",
//         "msg_pre": "https://account.masjingyao.top/notification/allMessage/detail?",
//         "msg_uri": "https://notifications.masjingyao.top/api/msg/v1/c/innermsg/list/unread/0?size=5&frt=0",
//         "name": "151****0103",
//         "nickname": "1235",
//         "page_id": "w3c00V9H4SeB3whvNbJmKwCBT-dra9ihmw4gem.account.p7",
//         "tavatar": "",
//         "tenant": "",
//         "trole_name": "",
//         "tuser_name": "",
//         "uri_auth": "https://authc.masjingyao.top/AuthenticationManagement/PersonalAuthentication",
//         "uri_debug": "?&papp=account&org=&dir=&user=0&role=0&kid=&redirect_uri=",
//         "uri_kanban": "https://account.masjingyao.top/kanban",
//         "uri_safe": "https://account.masjingyao.top/user/security",
//         "uri_user": "https://account.masjingyao.top/user/userinfo",
//         "uri_user_auth": "https://authc.masjingyao.top/AuthenticationManagement/PersonalAuthentication",
//         "user_name": "林*率",
//         "wiki_uri": "https://wiki.masjingyao.top/api/wiki/v1/search/popover?"
//     },
//     "traceId": "4de899c440132e0db18ad0b91af7810a"
// }
