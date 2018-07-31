/*Package mail 邮箱模块
一定要记得在confin.json配置这个模块的参数,否则无法使用
*/
package mail

import (
	"github.com/liangdas/mqant/conf"
	"github.com/liangdas/mqant/gate"
	"github.com/liangdas/mqant/module"
	"github.com/liangdas/mqant/module/base"
)

//Module 模块实例
var Module = func() module.Module {
	mail := new(Mail)
	return mail
}

//Mail 大厅结构体
type Mail struct {
	basemodule.BaseModule
}

//GetType 模块类型
func (mail *Mail) GetType() string {
	//很关键,需要与配置文件中的Module配置对应
	return "Mail"
}

//Version 模块版本
func (mail *Mail) Version() string {
	//可以在监控时了解代码版本
	return "1.0.0"
}

//OnInit 模块初始化
func (mail *Mail) OnInit(app module.App, settings *conf.ModuleSettings) {
	mail.BaseModule.OnInit(mail, app, settings)

	mail.GetServer().RegisterGO("HD_GetMailList", mail.getMailList)   //获取用户自己邮箱内消息，以及系统通知消息
	mail.GetServer().RegisterGO("HD_ReadUserMail", mail.readUserMail) //获取用户自己邮箱内消息，以及系统通知消息

	mail.GetServer().RegisterGO("HD_ReadSystemMail", mail.readSystemMail) //获取用户自己邮箱内消息，以及系统通知消息

}

//Run 模块运行
func (mail *Mail) Run(closeSig chan bool) {
}

//OnDestroy 模块清理
func (mail *Mail) OnDestroy() {
	//一定别忘了关闭RPC
	mail.GetServer().OnDestroy()
}

func (mail *Mail) getMailList(session gate.Session, msg []byte) (result module.ProtocolMarshal, err string) {

	retListString, err := mail.RpcInvoke("RedisDB", "getMailLists", session.GetUserid())

	return mail.App.ProtocolMarshal(retListString, err)
}

func (mail *Mail) readUserMail(session gate.Session, msg []byte) (result module.ProtocolMarshal, err string) {
	msgID := string(msg[1:])
	retString, err := mail.RpcInvoke("RedisDB", "readUserMail", session.GetUserid(), msgID)

	return mail.App.ProtocolMarshal(retString, err)
}

func (mail *Mail) readSystemMail(session gate.Session, msg []byte) (result module.ProtocolMarshal, err string) {
	msgID := string(msg[1:])
	retString, err := mail.RpcInvoke("RedisDB", "readSystemMail", session.GetUserid(), msgID)

	return mail.App.ProtocolMarshal(retString, err)
}
