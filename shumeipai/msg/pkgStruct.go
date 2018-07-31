//Package msg 客户端服务器通信结构体定义
package msg

//SCPackageInfo 服务端直接返回的数据是这个
type SCPackageInfo struct {
	Error  string
	Result string
}
