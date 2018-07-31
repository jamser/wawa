//Package webapp 后台模块
/**
一定要记得在confin.json配置这个模块的参数,否则无法使用
*/
package webapp

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"text/template"
	"time"

	"github.com/guidao/gopay"

	"github.com/gorilla/mux"
	"github.com/liangdas/mqant/conf"
	"github.com/liangdas/mqant/log"
	"github.com/liangdas/mqant/module"
	"github.com/liangdas/mqant/module/base"

	bgMsg "zhuawawa/msg"
)

//Module 模块实例
var Module = func() *Web {
	web := new(Web)
	return web
}

//Web 后台管理 结构体
type Web struct {
	basemodule.BaseModule
	staticPath      string
	DomainURL       string             //域名地址
	indextpl        *template.Template //index.html模板
	publicNoticetpl *template.Template //pub.html模板

	agenttpl     *template.Template //agent.html模板
	userInfotpl  *template.Template //registration.html模板
	tradeInfotpl *template.Template //recharge.html模板
	shoptpl      *template.Template //price.html
	devicetpl    *template.Template //equipment.html模板
	activitytpl  *template.Template //active.html模板
	wawatpl      *template.Template //unshipped.html模板
	contracttpl  *template.Template //contact.html模板
	/*	gameLobbytpl     *template.Template //gameLobby.htm模板
		ruletpl          *template.Template //rule.htm模板
		awardtpl         *template.Template //awardInfo.htm模板
		gameServerCancel context.CancelFunc //关闭游戏服务器
	*/
}

type userStruct struct {
	Nick string
	Pub  string
}

type passInfoStruct struct {
	User   userStruct
	Update bool   //是否显示编辑成功界面
	Des    string //成功编辑的提示
	Infos  map[string]interface{}
}

//GetType 模块类型
func (web *Web) GetType() string {
	//很关键,需要与配置文件中的Module配置对应
	return "Webapp"
}

//Version 模块版本
func (web *Web) Version() string {
	//可以在监控时了解代码版本
	return "1.0.0"
}

func genlist(start, end int) []int {

	ret := make([]int, end-start+1)
	for i := 0; i < len(ret); i++ {
		ret[i] = i + start
	}
	return ret
}

func getActiveMember(slice []bgMsg.SCActiveInfo, idx int) bgMsg.SCActiveInfo {
	return slice[idx]
	//	return "切片越界！"
}

func addOne(idx int) int {
	return idx + 1
}

//OnInit 模块初始化
func (web *Web) OnInit(app module.App, settings *conf.ModuleSettings) {
	web.BaseModule.OnInit(web, app, settings)
	web.staticPath = web.GetModuleSettings().Settings["StaticPath"].(string)
	web.DomainURL = web.GetModuleSettings().Settings["DomainURL"].(string)
	var err error
	web.indextpl, err = template.ParseFiles(web.staticPath + "/index.html")

	if err != nil {
		log.Error("index.html模板文件出错！！")
	}

	web.publicNoticetpl = template.New("pub")
	bytes, _ := ioutil.ReadFile(web.staticPath + "/pub.html") //读文件
	template.Must(web.publicNoticetpl.Parse(string(bytes)))   //将字符串读作模板

	web.agenttpl = template.New("agent")
	bytes, _ = ioutil.ReadFile(web.staticPath + "/agent.html") //读文件
	template.Must(web.agenttpl.Parse(string(bytes)))           //将字符串读作模板

	web.userInfotpl = template.New("userInfo").Funcs(template.FuncMap{"genList": genlist})
	bytes, _ = ioutil.ReadFile(web.staticPath + "/registration.html") //读文件
	template.Must(web.userInfotpl.Parse(string(bytes)))               //将字符串读作模板

	web.tradeInfotpl = template.New("tradeInfo").Funcs(template.FuncMap{"genList": genlist})
	bytes, _ = ioutil.ReadFile(web.staticPath + "/recharge.html") //读文件
	template.Must(web.tradeInfotpl.Parse(string(bytes)))          //将字符串读作模板

	web.shoptpl = template.New("shop")
	bytes, _ = ioutil.ReadFile(web.staticPath + "/price.html") //读文件
	template.Must(web.shoptpl.Parse(string(bytes)))            //将字符串读作模板

	web.devicetpl = template.New("device")
	bytes, _ = ioutil.ReadFile(web.staticPath + "/equipment.html") //读文件
	template.Must(web.devicetpl.Parse(string(bytes)))              //将字符串读作模板

	web.activitytpl = template.New("active").Funcs(template.FuncMap{"getActive": getActiveMember})
	bytes, _ = ioutil.ReadFile(web.staticPath + "/active.html") //读文件
	template.Must(web.activitytpl.Parse(string(bytes)))         //将字符串读作模板

	web.wawatpl = template.New("wawalist").Funcs(template.FuncMap{"AddOne": addOne, "genList": genlist})
	bytes, _ = ioutil.ReadFile(web.staticPath + "/unshipped.html") //读文件
	template.Must(web.wawatpl.Parse(string(bytes)))                //将字符串读作模板

	web.contracttpl = template.New("contact")
	bytes, _ = ioutil.ReadFile(web.staticPath + "/contact.html") //读文件
	template.Must(web.contracttpl.Parse(string(bytes)))          //将字符串读作模板

}

func loggingHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		//[26/Oct/2017:19:07:04 +0800]`-`"GET /g/c HTTP/1.1"`"curl/7.51.0"`502`[127.0.0.1]`-`"-"`0.006`166`-`-`127.0.0.1:8030`-`0.000`xd
		log.Info("%s %s %s [%s] in %v", r.Method, r.URL.Path, r.Proto, r.RemoteAddr, time.Since(start))
	})
}

//注销返回登陆界面
func (web *Web) signOut(w http.ResponseWriter, r *http.Request) {

	uidCookie, _ := r.Cookie("uid")
	if uidCookie != nil {
		uidCookie.Expires = time.Now().AddDate(-1, 0, 0)
	}
	http.SetCookie(w, uidCookie)

	tmpl, _ := web.indextpl.Clone()
	//tmpl, _ := template.ParseFiles(web.staticPath + "/index.html")

	tmpl.Execute(w, nil)
	return
}

//登陆检测处理函数
func (web *Web) logonHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		uName := r.PostFormValue("account")
		pwd := r.PostFormValue("pwd")
		h := md5.New()
		h.Write([]byte(pwd)) // 需要加密的字符串
		cipherStr := h.Sum(nil)
		bRet, err := web.RpcInvoke("RedisDB", "loginBG", uName, hex.EncodeToString(cipherStr))

		if err != "" {
			log.Error("后台登陆出错！%s ： %s", uName, err)
		}
		retTable := bRet.(map[string]interface{})

		if retTable["state"].(string) == "ok" {
			//log.Info("用户%s登陆后台！", uName)

			//设置cookie，之后从cookie中获取用户ID，若不存在则说明未登陆
			c1 := http.Cookie{
				Name:     "uid",
				Value:    uName,
				HttpOnly: true,
				Expires:  time.Now().AddDate(1, 0, 0),
			}

			http.SetCookie(w, &c1)

			/*tmpl := template.New("Main").Funcs(template.FuncMap{"Select": selectMenu})

			bytes, _ := ioutil.ReadFile(web.staticPath + "/main.html") //读文件
			template.Must(tmpl.Parse(string(bytes)))                   //将字符串读作模板
			ld := loginDefault{State: 2}
			tmpl.Execute(w, ld)
			*/
			//总代理需要查看设备信息

			tplData := passInfoStruct{User: userStruct{Nick: retTable["nick"].(string), Pub: retTable["pub"].(string)}, Update: false}

			/*
				tmpl := template.New("Main")
				bytes, _ := ioutil.ReadFile(web.staticPath + "/pub.html") //读文件
				template.Must(tmpl.Parse(string(bytes)))                  //将字符串读作模板
			*/
			tmpl, _ := web.publicNoticetpl.Clone()

			tmpl.Execute(w, tplData)
			return
		}
	}

	log.Warning("跳转有误！！！请程序查看")

	//tmpl, _ := template.ParseFiles(web.staticPath + "/index.html")
	tmpl, _ := web.indextpl.Clone()
	tmpl.Execute(w, nil)
	return
}

func (web *Web) pubNoticeHandler(w http.ResponseWriter, r *http.Request) {

	uidCookie, _ := r.Cookie("uid")

	bRet, err := web.RpcInvoke("RedisDB", "pubBG", uidCookie.Value)
	if err != "" {
		log.Error("跳转公告设置界面出错！%s ： %s", uidCookie.Value, err)
	}
	retTable := bRet.(map[string]interface{})
	tplData := passInfoStruct{User: userStruct{Nick: retTable["nick"].(string), Pub: retTable["pub"].(string)}, Update: false}

	tmpl, _ := web.publicNoticetpl.Clone()

	tmpl.Execute(w, tplData)

	return
}

func (web *Web) changePub(w http.ResponseWriter, r *http.Request) {

	uidCookie, _ := r.Cookie("uid")

	if uidCookie != nil && r.Method == "POST" {
		newPub := r.PostFormValue("newPub")
		//log.Info("更改公告为：%s", newPub)

		bRet, err := web.RpcInvoke("RedisDB", "changePub", uidCookie.Value, newPub)
		if err != "" {
			log.Error("更改公告出错！%s ： %s", uidCookie.Value, err)
		}
		retTable := bRet.(map[string]interface{})
		tplData := passInfoStruct{User: userStruct{Nick: retTable["nick"].(string), Pub: retTable["pub"].(string)},
			Update: true, Des: "大厅公告修改成功！"}
		tmpl, _ := web.publicNoticetpl.Clone()

		exeErr := tmpl.Execute(w, tplData)
		if exeErr != nil {
			log.Info("模板错误！：%s", exeErr.Error())
		}
	}

	return
}

func (web *Web) active(w http.ResponseWriter, r *http.Request) {

	uidCookie, _ := r.Cookie("uid")

	bRet, err := web.RpcInvoke("RedisDB", "getActives")
	if err != "" {
		log.Error("活动设置界面出错！%s ： %s", uidCookie.Value, err)
	}

	retData := new(bgMsg.SCActivesInfo)
	retData.UnmarshalJSON([]byte(bRet.(string)))

	tplData := passInfoStruct{User: userStruct{Nick: "管理员"}, Infos: map[string]interface{}{"Actives": retData}, Update: false}

	tmpl, _ := web.activitytpl.Clone()

	tmpl.Execute(w, tplData)

	return
}

func (web *Web) changeActive(w http.ResponseWriter, r *http.Request) {

	uidCookie, _ := r.Cookie("uid")

	r.ParseMultipartForm(32 >> 20)

	if uidCookie != nil && r.Method == "POST" {
		log.Info("id is %s", r.MultipartForm.Value["id"][0])
		newActive := new(bgMsg.SCActiveInfo)
		switch r.MultipartForm.Value["id"][0] {
		case "活动1":
			newActive.ActiveID = 1
		case "活动2":
			newActive.ActiveID = 2
		case "活动3":
			newActive.ActiveID = 3
		case "活动4":
			newActive.ActiveID = 4
		}

		newActive.ImgURL = fmt.Sprintf("%s/Imgactive/%d.png", web.DomainURL, newActive.ActiveID)
		if len(r.MultipartForm.File) != 0 {
			newFile := r.MultipartForm.File["update"][0]

			f, _ := os.OpenFile(fmt.Sprintf("%s/Imgactive/%d.png", web.staticPath, newActive.ActiveID), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, os.ModePerm)
			defer f.Close()
			t, _ := newFile.Open()
			io.Copy(f, t)

			//log.Info("上传文件的大小为: %d -- %s", newFile.Size, newFile.Filename)

		}

		buf, _ := newActive.MarshalJSON()
		_, err := web.RpcInvoke("RedisDB", "changeActive", string(buf))
		if err != "" {
			log.Error("修改活动信息出错！ %s", err)
		}

		http.Redirect(w, r, "/active", http.StatusFound)
	}

	return
}

func (web *Web) agent(w http.ResponseWriter, r *http.Request) {

	uidCookie, _ := r.Cookie("uid")

	if uidCookie != nil {

		bRet, err := web.RpcInvoke("RedisDB", "getInviteConfig")
		if err != "" {
			log.Error("获取邀请奖励信息出错！ %s", err)
		}
		retTable := bRet.(map[string]interface{})

		tplData := passInfoStruct{User: userStruct{Nick: "管理员"},
			Infos: map[string]interface{}{"InviteConfig": retTable}, Update: false}
		tmpl, _ := web.agenttpl.Clone()

		exeErr := tmpl.Execute(w, tplData)
		if exeErr != nil {
			log.Info("模板错误！：%s", exeErr.Error())
		}
	}

	return
}

func (web *Web) upAgent(w http.ResponseWriter, r *http.Request) {

	uidCookie, _ := r.Cookie("uid")
	//log.Info("修改新大厅：%v", uidCookie)

	if uidCookie != nil && r.Method == "POST" {
		r.ParseForm()
		key, vaule := "", ""
		for k, v := range r.PostForm {
			key = k
			vaule = v[0]
		}

		//log.Info("修改邀请奖励  %s：%s", key, vaule)
		bRet, err := web.RpcInvoke("RedisDB", "changeInviteConfig", key, vaule)
		if err != "" {
			log.Error("修改邀请奖励配置信息出错！ %s", err)
		}
		retTable := bRet.(map[string]interface{})

		tplData := passInfoStruct{User: userStruct{Nick: "管理员"},
			Infos: map[string]interface{}{"InviteConfig": retTable}, Update: false}
		tmpl, _ := web.agenttpl.Clone()

		exeErr := tmpl.Execute(w, tplData)
		if exeErr != nil {
			log.Info("模板错误！：%s", exeErr.Error())
		}
	}

	return
}

func calPagesInfo(startIdx, endIdx, curPageIdx, totalPage int) (left, right bool, firstIdx, lastIdx, curIdx int) {
	if curPageIdx >= totalPage {
		//log.Info("选中最后一页！")
		//选中最后一页
		firstIdx = totalPage - 4
		lastIdx = totalPage
	} else if curPageIdx == endIdx {
		//点击最后一个数字选项卡，更新start与end的idx
		//log.Info("选中最后一个选项卡")
		firstIdx = endIdx
		lastIdx = endIdx + 4
	} else if curPageIdx == startIdx {
		//点击第一个数字选项卡，更新start与end的idx
		//log.Info("选中第一个选项卡")
		lastIdx = startIdx
		firstIdx = endIdx - 4
	}

	if lastIdx >= totalPage {
		lastIdx = totalPage
		firstIdx = lastIdx - 4
		//right = false
	}

	if firstIdx <= 1 {
		firstIdx = 1
		lastIdx = 5
		if lastIdx > totalPage {
			lastIdx = totalPage
		}
		//left = false
	}

	if curPageIdx > lastIdx {
		curIdx = lastIdx
	} else if curPageIdx < firstIdx {
		curIdx = firstIdx
	} else {
		curIdx = curPageIdx
	}

	left, right = true, true
	if curIdx == firstIdx {
		left = false
	}

	if curIdx == lastIdx {
		right = false
	}
	return
}

func generateAwardStruct(mailInfo, start, end string, bUnSend bool) passInfoStruct {
	retUserList := new(bgMsg.SCBGUserInfoSum)
	retUserList.UnmarshalJSON([]byte(mailInfo))

	iStardPage, _ := strconv.Atoi(start)
	iEndPage, _ := strconv.Atoi(end)
	left, right, firstIdx, lastIdx, curIdx := calPagesInfo(iStardPage, iEndPage, retUserList.CurPage, retUserList.TotalPage)

	tplData := passInfoStruct{User: userStruct{Nick: "管理员"},
		Infos: map[string]interface{}{"UserList": retUserList, "Left": left, "Right": right,
			"First": firstIdx, "Last": lastIdx, "CurPage": curIdx}, Update: false}
	return tplData
}

func (web *Web) userInfo(w http.ResponseWriter, r *http.Request) {

	uidCookie, _ := r.Cookie("uid")

	if uidCookie != nil {
		if r.Method == "GET" {
			userInfo, err := web.RpcInvoke("RedisDB", "getBGUserInfo", r.FormValue("curPage"))

			if err != "" {
				log.Error("%s", err)
			}

			tmpl, _ := web.userInfotpl.Clone()
			tplData := generateAwardStruct(userInfo.(string), r.FormValue("startPage"), r.FormValue("endPage"), true)
			exeErr := tmpl.Execute(w, tplData)
			if exeErr != nil {
				log.Info("模板错误！：%s", exeErr.Error())
			}
		}
	}
	return
}

func (web *Web) reward(w http.ResponseWriter, r *http.Request) {

	uidCookie, _ := r.Cookie("uid")

	if uidCookie != nil {
		if r.Method == "GET" {
			recordInfo, err := web.RpcInvoke("RedisDB", "getBGUnsendList", r.FormValue("curPage"))

			if err != "" {
				log.Error("%s", err)
			}

			tmpl, _ := web.wawatpl.Clone()
			//tplData := generateAwardStruct(recordInfo, r.FormValue("startPage"), r.FormValue("endPage"), true)
			retRecordList := new(bgMsg.SCBGRecordSum)
			retRecordList.UnmarshalJSON([]byte(recordInfo.(string)))

			iStardPage, _ := strconv.Atoi(r.FormValue("startPage"))
			iEndPage, _ := strconv.Atoi(r.FormValue("endPage"))
			left, right, firstIdx, lastIdx, curIdx := calPagesInfo(iStardPage, iEndPage, retRecordList.CurPage, retRecordList.TotalPage)

			tplData := passInfoStruct{User: userStruct{Nick: "管理员"},
				Infos: map[string]interface{}{"RecordList": retRecordList, "Left": left, "Right": right,
					"First": firstIdx, "Last": lastIdx, "CurPage": curIdx, "Type": 1}, Update: false} //Type 1: 中奖列表
			exeErr := tmpl.Execute(w, tplData)
			if exeErr != nil {
				log.Info("模板错误！：%s", exeErr.Error())
			}
		}
	}
	return
}

func (web *Web) askSend(w http.ResponseWriter, r *http.Request) {

	uidCookie, _ := r.Cookie("uid")

	if uidCookie != nil {
		if r.Method == "GET" {
			recordInfo, err := web.RpcInvoke("RedisDB", "getBGAsksendList", r.FormValue("curPage"))

			if err != "" {
				log.Error("%s", err)
			}

			tmpl, _ := web.wawatpl.Clone()
			//tplData := generateAwardStruct(recordInfo, r.FormValue("startPage"), r.FormValue("endPage"), true)
			retRecordList := new(bgMsg.SCBGRecordSum)
			retRecordList.UnmarshalJSON([]byte(recordInfo.(string)))

			iStardPage, _ := strconv.Atoi(r.FormValue("startPage"))
			iEndPage, _ := strconv.Atoi(r.FormValue("endPage"))
			left, right, firstIdx, lastIdx, curIdx := calPagesInfo(iStardPage, iEndPage, retRecordList.CurPage, retRecordList.TotalPage)

			tplData := passInfoStruct{User: userStruct{Nick: "管理员"},
				Infos: map[string]interface{}{"RecordList": retRecordList, "Left": left, "Right": right,
					"First": firstIdx, "Last": lastIdx, "CurPage": curIdx, "Type": 2}, Update: false} //Type 1: 中奖列表
			exeErr := tmpl.Execute(w, tplData)
			if exeErr != nil {
				log.Info("模板错误！：%s", exeErr.Error())
			}
		}
	}
	return
}

func (web *Web) sended(w http.ResponseWriter, r *http.Request) {

	uidCookie, _ := r.Cookie("uid")

	if uidCookie != nil {
		if r.Method == "GET" {
			recordInfo, err := web.RpcInvoke("RedisDB", "getBGSendedList", r.FormValue("curPage"))

			if err != "" {
				log.Error("%s", err)
			}

			tmpl, _ := web.wawatpl.Clone()
			//tplData := generateAwardStruct(recordInfo, r.FormValue("startPage"), r.FormValue("endPage"), true)
			retRecordList := new(bgMsg.SCBGRecordSum)
			retRecordList.UnmarshalJSON([]byte(recordInfo.(string)))

			iStardPage, _ := strconv.Atoi(r.FormValue("startPage"))
			iEndPage, _ := strconv.Atoi(r.FormValue("endPage"))
			left, right, firstIdx, lastIdx, curIdx := calPagesInfo(iStardPage, iEndPage, retRecordList.CurPage, retRecordList.TotalPage)

			tplData := passInfoStruct{User: userStruct{Nick: "管理员"},
				Infos: map[string]interface{}{"RecordList": retRecordList, "Left": left, "Right": right,
					"First": firstIdx, "Last": lastIdx, "CurPage": curIdx, "Type": 3}, Update: false} //Type 1: 中奖列表
			exeErr := tmpl.Execute(w, tplData)
			if exeErr != nil {
				log.Info("模板错误！：%s", exeErr.Error())
			}
		}
	}
	return
}

func (web *Web) sendPrize(w http.ResponseWriter, r *http.Request) {

	uidCookie, _ := r.Cookie("uid")

	if uidCookie != nil {
		if r.Method == "GET" {
			_, err := web.RpcInvoke("RedisDB", "snedPrize", r.FormValue("askID"), r.FormValue("curPage"), r.FormValue("uID"))

			if err != "" {
				log.Error("%s", err)
			}

			http.Redirect(w, r, fmt.Sprintf("/askSend?startPage=%s&endPage=%s&curPage=%s", r.FormValue("startPage"),
				r.FormValue("endPage"), r.FormValue("curPage")), http.StatusFound)
		}
	}
}

func (web *Web) shopInfos(w http.ResponseWriter, r *http.Request) {

	uidCookie, _ := r.Cookie("uid")

	if uidCookie != nil {

		bRet, err := web.RpcInvoke("RedisDB", "getShopList")
		if err != "" {
			log.Error("获取商品信息出错！ %s", err)
		}
		shopInfos := new(bgMsg.SCAllShopItems)
		shopInfos.UnmarshalJSON([]byte(bRet.(string)))

		tplData := passInfoStruct{User: userStruct{Nick: "管理员"},
			Infos: map[string]interface{}{"ShopInfos": shopInfos}, Update: false}
		tmpl, _ := web.shoptpl.Clone()
		exeErr := tmpl.Execute(w, tplData)
		if exeErr != nil {
			log.Info("模板错误！：%s", exeErr.Error())
		}
	}

	return
}

func (web *Web) configShop(w http.ResponseWriter, r *http.Request) {

	uidCookie, _ := r.Cookie("uid")
	r.ParseMultipartForm(32 >> 20)

	if uidCookie != nil && r.Method == "POST" {

		newShopItem := new(bgMsg.SCShopItemInfo)
		newShopItem.ID = r.MultipartForm.Value["id"][0]
		newShopItem.Title = r.MultipartForm.Value["name"][0]
		newShopItem.ShopDes = r.MultipartForm.Value["introduce"][0]
		newShopItem.Cost, _ = strconv.Atoi(r.MultipartForm.Value["amount"][0])
		newShopItem.Gold, _ = strconv.Atoi(r.MultipartForm.Value["obtained"][0])
		newShopItem.ExtraGot, _ = strconv.Atoi(r.MultipartForm.Value["date_obtained"][0])
		if r.MultipartForm.Value["type"][0] == "月卡" {
			newShopItem.Type = 2
		} else if r.MultipartForm.Value["type"][0] == "周卡" {
			newShopItem.Type = 1
		} else if r.MultipartForm.Value["type"][0] == "普通充值" {
			newShopItem.Type = 0
		}

		newShopItem.ImageURL = fmt.Sprintf("%s/shopIcons/%s.png", web.DomainURL, newShopItem.ID)
		if len(r.MultipartForm.File) != 0 {
			newFile := r.MultipartForm.File["update"][0]

			f, _ := os.OpenFile(fmt.Sprintf("%s/shopIcons/%s.png", web.staticPath, newShopItem.ID), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, os.ModePerm)
			defer f.Close()
			t, _ := newFile.Open()
			io.Copy(f, t)

			//log.Info("上传文件的大小为: %d -- %s", newFile.Size, newFile.Filename)

		}

		buf, _ := newShopItem.MarshalJSON()
		_, err := web.RpcInvoke("RedisDB", "changeShopConfig", buf, newShopItem.ID)
		if err != "" {
			log.Error("修改商品信息出错！ %s", err)
		}
		http.Redirect(w, r, "/shopInfos", http.StatusFound)
	}

	return
}

func (web *Web) equip(w http.ResponseWriter, r *http.Request) {

	uidCookie, _ := r.Cookie("uid")

	if uidCookie != nil {

		bRet, err := web.RpcInvoke("RedisDB", "getDevicesInfo")
		if err != "" {
			log.Error("获取商品信息出错！ %s", err)
		}
		devices := new(bgMsg.SCAllDevices)
		devices.UnmarshalJSON(bRet.([]byte))

		tplData := passInfoStruct{User: userStruct{Nick: "管理员"},
			Infos: map[string]interface{}{"DeviceInfos": devices}, Update: false}
		tmpl, _ := web.devicetpl.Clone()
		exeErr := tmpl.Execute(w, tplData)
		if exeErr != nil {
			log.Info("模板错误！：%s", exeErr.Error())
		}
	}

	return
}

func (web *Web) configEquip(w http.ResponseWriter, r *http.Request) {

	uidCookie, _ := r.Cookie("uid")
	r.ParseMultipartForm(32 >> 20)

	if uidCookie != nil && r.Method == "POST" {

		newDevice := new(bgMsg.SCDeviceInfo)
		newDevice.DeviceID = r.MultipartForm.Value["id"][0]
		newDevice.Destription = r.MultipartForm.Value["name"][0]
		newDevice.Cost, _ = strconv.Atoi(r.MultipartForm.Value["price"][0])
		newDevice.LeftCount, _ = strconv.Atoi(r.MultipartForm.Value["remainder"][0])
		newDevice.Force, _ = strconv.Atoi(r.MultipartForm.Value["power"][0])
		newDevice.Group, _ = strconv.Atoi(r.MultipartForm.Value["type"][0])
		newDevice.Exchange, _ = strconv.Atoi(r.MultipartForm.Value["exchange"][0])

		newDevice.DesImg = fmt.Sprintf("%s/DevIcons/%s-Des.png", web.DomainURL, newDevice.DeviceID)
		newDevice.Thumbnail = fmt.Sprintf("%s/DevIcons/%s-Tub.png", web.DomainURL, newDevice.DeviceID)

		if newDes, bOK := r.MultipartForm.File["updescribe"]; bOK {
			f, _ := os.OpenFile(fmt.Sprintf("%s/DevIcons/%s-Des.png", web.staticPath, newDevice.DeviceID), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, os.ModePerm)
			defer f.Close()
			t, _ := newDes[0].Open()
			io.Copy(f, t)
			//log.Info("描述图上传文件的大小为: %d -- %s", newDes[0].Size, newDes[0].Filename)
		}

		if newDes, bOK := r.MultipartForm.File["updateTub"]; bOK {
			f, _ := os.OpenFile(fmt.Sprintf("%s/DevIcons/%s-Tub.png", web.staticPath, newDevice.DeviceID), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, os.ModePerm)
			defer f.Close()
			t, _ := newDes[0].Open()
			io.Copy(f, t)
			//log.Info("缩略图上传文件的大小为: %d -- %s", newDes[0].Size, newDes[0].Filename)
		}

		buf, _ := newDevice.MarshalJSON()
		_, err := web.RpcInvoke("RedisDB", "changeDeviceConfig", buf)
		if err != "" {
			log.Error("修改设备信息出错！ %s", err)
		}
		http.Redirect(w, r, "/equipment", http.StatusFound)
	}

	return
}

func (web *Web) payNotify(w http.ResponseWriter, r *http.Request) {
	wechatResult, err := gopay.WeChatH5Callback(w, r)
	if err != nil {
		log.Warning(err.Error())
		return
	}
	//log.Info("支付结果通知： %v", wechatResult)
	if wechatResult.ResultCode == "SUCCESS" {
		log.Info("成功通知接收到了:%s", wechatResult.OutTradeNO)
		web.RpcInvokeNR("RedisDB", "buyDone", wechatResult.OutTradeNO, true, int64(2))
	} else if wechatResult.ResultCode == "FAIL" {
		log.Info("失败通知接收到了:%s", wechatResult.OutTradeNO)
		web.RpcInvokeNR("RedisDB", "buyDone", wechatResult.OutTradeNO, true, int64(3))
	}
	return
}

func (web *Web) tradeInfo(w http.ResponseWriter, r *http.Request) {

	uidCookie, _ := r.Cookie("uid")

	if uidCookie != nil {
		if r.Method == "GET" {
			tradeInfos, err := web.RpcInvoke("RedisDB", "getBGTradesInfo", r.FormValue("curPage"))

			if err != "" {
				log.Error("%s", err)
			}

			tmpl, _ := web.tradeInfotpl.Clone()
			retTradeList := new(bgMsg.SCAllTradeInfos)
			retTradeList.UnmarshalJSON([]byte(tradeInfos.(string)))

			iStardPage, _ := strconv.Atoi(r.FormValue("startPage"))
			iEndPage, _ := strconv.Atoi(r.FormValue("endPage"))
			left, right, firstIdx, lastIdx, curIdx := calPagesInfo(iStardPage, iEndPage, retTradeList.CurPage, retTradeList.TotalPage)

			tplData := passInfoStruct{User: userStruct{Nick: "管理员"},
				Infos: map[string]interface{}{"TradeList": retTradeList, "Left": left, "Right": right,
					"First": firstIdx, "Last": lastIdx, "CurPage": curIdx}, Update: false}
			//tplData := generateAwardStruct(userInfo.(string), r.FormValue("startPage"), r.FormValue("endPage"), true)
			exeErr := tmpl.Execute(w, tplData)
			if exeErr != nil {
				log.Info("模板错误！：%s", exeErr.Error())
			}
		}
	}
	return
}

func (web *Web) contact(w http.ResponseWriter, r *http.Request) {

	uidCookie, _ := r.Cookie("uid")

	if uidCookie != nil {
		if r.Method == "GET" {
			tradeInfos, err := web.RpcInvoke("RedisDB", "getBGContackInfo")

			if err != "" {
				log.Error("%s", err)
			}

			tmpl, _ := web.contracttpl.Clone()
			retContactInfo := new(bgMsg.SCContactInfo)
			retContactInfo.UnmarshalJSON([]byte(tradeInfos.(string)))

			tplData := passInfoStruct{User: userStruct{Nick: "管理员"},
				Infos: map[string]interface{}{"Contact": retContactInfo}, Update: false}

			exeErr := tmpl.Execute(w, tplData)
			if exeErr != nil {
				log.Info("模板错误！：%s", exeErr.Error())
			}
		}
	}
	return
}

func (web *Web) upQQ(w http.ResponseWriter, r *http.Request) {

	uidCookie, _ := r.Cookie("uid")

	if uidCookie != nil {
		if r.Method == "POST" {
			newQQ := r.FormValue("qq")
			if newQQ != "" {
				web.RpcInvoke("RedisDB", "bgUpQQ", newQQ)
			}
			http.Redirect(w, r, "/contact", http.StatusFound)
		}
	}
	return
}

func (web *Web) upWechat(w http.ResponseWriter, r *http.Request) {

	uidCookie, _ := r.Cookie("uid")

	if uidCookie != nil {
		if r.Method == "POST" {
			newWechat := r.FormValue("wechat")
			if newWechat != "" {
				web.RpcInvoke("RedisDB", "bgUpWechat", newWechat)
			}
			http.Redirect(w, r, "/contact", http.StatusFound)
		}
	}
	return
}

func (web *Web) upKeFu(w http.ResponseWriter, r *http.Request) {

	uidCookie, _ := r.Cookie("uid")

	if uidCookie != nil {
		if r.Method == "POST" {
			newKeFu := r.FormValue("kefu")
			if newKeFu != "" {
				web.RpcInvoke("RedisDB", "bgUpKeFu", newKeFu)
			}
			http.Redirect(w, r, "/contact", http.StatusFound)
		}
	}
	return
}

//Run 模块运行
func (web *Web) Run(closeSig chan bool) {
	//这里如果出现异常请检查8080端口是否已经被占用
	l, err := net.Listen("tcp", ":3500")
	if err != nil {
		log.Error("webapp server error", err.Error())
		return
	}
	log.Info("webapp server Listen : %s", ":3500")
	root := mux.NewRouter()
	//注销后返回登陆页面
	status := root.PathPrefix("/signOut")
	status.HandlerFunc(web.signOut)
	//登陆后前往公告设置界面
	status = root.PathPrefix("/login")
	status.HandlerFunc(web.logonHandler)
	//点击头部导航栏前往公告设置界面
	status = root.PathPrefix("/pub")
	status.HandlerFunc(web.pubNoticeHandler)
	//修改公告设置
	status = root.PathPrefix("/upPub")
	status.HandlerFunc(web.changePub)

	//点击头部导航栏前往活动设置界面
	status = root.PathPrefix("/active")
	status.HandlerFunc(web.active)
	//修改公告设置
	status = root.PathPrefix("/configActives")
	status.HandlerFunc(web.changeActive)

	//前往邀请奖励设置界面
	status = root.PathPrefix("/agent")
	status.HandlerFunc(web.agent)
	//修改邀请奖励设置
	status = root.PathPrefix("/upAgent")
	status.HandlerFunc(web.upAgent)

	//前往玩家信息界面
	status = root.PathPrefix("/userinfo")
	status.HandlerFunc(web.userInfo)

	//前往商品信息界面
	status = root.PathPrefix("/shopInfos")
	status.HandlerFunc(web.shopInfos)
	//修改商品信息界面
	status = root.PathPrefix("/configShop")
	status.HandlerFunc(web.configShop)

	//前往设备信息设置界面
	status = root.PathPrefix("/equipment")
	status.HandlerFunc(web.equip)
	//修改设备信息规则界面
	status = root.PathPrefix("/configEquip")
	status.HandlerFunc(web.configEquip)

	//处理发货信息
	status = root.PathPrefix("/reward")
	status.HandlerFunc(web.reward)

	status = root.PathPrefix("/askSend")
	status.HandlerFunc(web.askSend)

	status = root.PathPrefix("/sended")
	status.HandlerFunc(web.sended)

	//处理发货信息
	status = root.PathPrefix("/sendPrize")
	status.HandlerFunc(web.sendPrize)

	//H5支付结果跳转
	status = root.PathPrefix("/payNotify")
	status.HandlerFunc(web.payNotify)

	//订单信息
	status = root.PathPrefix("/tradeInfo")
	status.HandlerFunc(web.tradeInfo)

	//联系信息
	status = root.PathPrefix("/contact")
	status.HandlerFunc(web.contact)
	//联系信息
	status = root.PathPrefix("/upQQ")
	status.HandlerFunc(web.upQQ)
	//联系信息
	status = root.PathPrefix("/upWechat")
	status.HandlerFunc(web.upWechat)
	//联系信息
	status = root.PathPrefix("/upKeFu")
	status.HandlerFunc(web.upKeFu)
	/*
			status = root.PathPrefix("/awardUnSend")
		status.HandlerFunc(web.awardPage)

		//中奖信息界面(显示已发货)
		status = root.PathPrefix("/awardSended")
		status.HandlerFunc(web.awardPageSend)
	*/
	static := root.PathPrefix("/")
	static.Handler(http.StripPrefix("/", http.FileServer(http.Dir(web.staticPath))))
	//r.Handle("/static",static)
	ServeMux := http.NewServeMux()
	ServeMux.Handle("/", root)
	http.Serve(l, loggingHandler(ServeMux))

	<-closeSig
	log.Info("webapp server Shutting down...")
	l.Close()
}

//OnDestroy 模块清理
func (web *Web) OnDestroy() {
	//一定别忘了关闭RPC
	web.GetServer().OnDestroy()
}
