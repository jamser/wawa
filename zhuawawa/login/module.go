/*Package login  登陆模块，负责客户端注册登陆事宜
一定要记得在confin.json配置这个模块的参数,否则无法使用
*/
package login

import (
	loginMsg "zhuawawa/msg"

	"github.com/liangdas/mqant/conf"
	"github.com/liangdas/mqant/gate"
	"github.com/liangdas/mqant/log"
	"github.com/liangdas/mqant/module"
	"github.com/liangdas/mqant/module/base"
)

//Module 模块实例
var Module = func() module.Module {
	loginModule := new(Login)
	return loginModule
}

//Login 登陆模块结构体
type Login struct {
	basemodule.BaseModule
	IOSAppStoreVersion bool //是否AppStore审核版本
}

//GetType 模块类型
func (m *Login) GetType() string {
	//很关键,需要与配置文件中的Module配置对应
	return "Login"
}

//Version 模块版本
func (m *Login) Version() string {
	//可以在监控时了解代码版本
	return "1.0.0"
}

//OnInit 模块初始化
func (m *Login) OnInit(app module.App, settings *conf.ModuleSettings) {
	m.BaseModule.OnInit(m, app, settings)

	m.GetServer().RegisterGO("HD_Login", m.login)       //我们约定所有对客户端的请求都以Handler_开头（客户端请求登陆）
	m.GetServer().RegisterGO("HD_Register", m.register) //我们约定所有对客户端的请求都以Handler_开头（客户端请求注册）
	m.IOSAppStoreVersion = m.GetModuleSettings().Settings["appStore"].(bool)
}

//Run 模块运行
func (m *Login) Run(closeSig chan bool) {
}

//OnDestroy 模块清理
func (m *Login) OnDestroy() {
	//一定别忘了关闭RPC
	m.GetServer().OnDestroy()
}

//登陆模块
func (m *Login) login(session gate.Session, msg []byte) (result module.ProtocolMarshal, err string) {
	loginmsg := new(loginMsg.CSLogin)
	jsonErr := loginmsg.UnmarshalJSON(msg)
	log.Info("user %s logining ... ", loginmsg.Username)
	retMsg := loginMsg.SCLoginRet{Success: false, Register: false}

	if jsonErr != nil {
		retMsg.Description = "登陆信息格式有误！"
		retData, _ := retMsg.MarshalJSON()
		return m.App.ProtocolMarshal(string(retData), "")
	}
	if loginmsg.Username == "" || loginmsg.Password == "" {
		retMsg.Description = "登陆用户名或者密码为空"
		retData, _ := retMsg.MarshalJSON()
		return m.App.ProtocolMarshal(string(retData), "")
	}

	ret, err := m.RpcInvoke("RedisDB", "checklogin", loginmsg.Username, loginmsg.Password)

	if err != "" {
		log.Info("check login err: %s", err)
		return
	}

	loginRet, bOk := ret.(map[string]interface{})
	if bOk {
		if loginRet["register"] != nil {
			//需要注册
			retMsg.Register = true
			retMsg.Description = "账号不存在，请注册后重新登陆"
			retData, _ := retMsg.MarshalJSON()
			return m.App.ProtocolMarshal(string(retData), "")
		}

		if loginRet["success"].(bool) {
			retMsg.Success = true
			retMsg.Description = "登陆成功！"
			retMsg.PublicNotice = loginRet["pub"].(string)
			retMsg.IOSAppStore = m.IOSAppStoreVersion
		} else {
			retMsg.Description = "登陆失败！"
		}

	} else {
		log.Warning("登陆返回值有误")
	}

	if retMsg.Success {
		if session.GetUserid() == "" {
			err = session.Bind(loginmsg.Username)
			if err != "" {
				return
			}
		}

		//綁定成功，此时session已从redis中读取了最新的数据
		retMsg.LastPlace = session.GetSettings()["lastplace"]
	}
	retData, _ := retMsg.MarshalJSON()
	//log.Info("user %s logining ... with retdes len %d  :  %s", loginRet["success"], len(retData), string(retData))
	return m.App.ProtocolMarshal(string(retData), "")
}

//注册模块
func (m *Login) register(session gate.Session, msg []byte) (result module.ProtocolMarshal, err string) {
	registermsg := new(loginMsg.CSRegister)
	jsonErr := registermsg.UnmarshalJSON(msg)
	log.Info("user %s registering ... ", registermsg.Username)

	if jsonErr != nil {
		err = "注册信息格式有误！"
		return m.App.ProtocolMarshal("", err)
	}
	if registermsg.Username == "" || registermsg.Password == "" {
		err = "登陆用户名或者密码为空"
		return m.App.ProtocolMarshal("", err)
	}

	_, err = m.RpcInvoke("RedisDB", "register", session, msg)

	if err != "" {
		log.Info("register err: %s", err)
		return
	}

	return m.App.ProtocolMarshal("", err)
}
