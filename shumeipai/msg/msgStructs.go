//Package msg 客户端服务器通信结构体定义
package msg

//CSLogin 客户端登陆结构体
type CSLogin struct {
	Username string `json:"account"` //账号
	Password string `json:"pwd"`     //密码
}

//SCLoginRet 服务端登陆返回结构体
type SCLoginRet struct {
	Success      bool   `json:"success"`     //登陆是否成功
	Register     bool   `json:"register"`    //是否需要注册
	LastPlace    string `json:"lastplace"`   //上次离线所在模块
	Description  string `json:"description"` //登陆返回信息
	PublicNotice string `json:"pub"`         //公告信息
	IOSAppStore  bool   `json:"iosStore"`    //是否AppStore审核版本
}

//CSRegister 客户端注册结构体
type CSRegister struct {
	Username string `json:"account"` //账号
	Password string `json:"pwd"`     //密码
	NickName string `json:"nick"`    //昵称
	HeadURL  string `json:"head"`    //头像URL
	Gender   bool   `json:"sex"`     //性别
	OpenID   string `json:"openid"`  //微信openID， H5支付用
}

//SCUserInfoRet 服务器用户信息返回
type SCUserInfoRet struct {
	ID       int64  `json:"id"`   //用户ID
	NickName string `json:"nick"` //昵称
	HeadURL  string `json:"head"` //头像URL
	Gender   bool   `json:"sex"`  //性别
	Gold     int64  `json:"gold"` //金币
}

//SCTimeUpdate 倒计时更新
type SCTimeUpdate struct {
	TimeCountDown int64 `json:"time"` //倒计时
}

//SCDeviceInfo 娃娃机设备信息
type SCDeviceInfo struct {
	State       int    `json:"state"`     //设备状态  -1 设备下线 0： 设备可玩 1： 设备正在运行, 2 :设备下抓中 3：中奖判定中 4:中奖  5：未中奖
	Group       int    `json:"group"`     //设备所属类型（新品， 热门，特价， 实用品等，二进制与判断相应位是否为1）
	SubGroup    int    `json:"sub"`       //设备子类型（同款列表显示用，不重复，从0开始每种类型加一）
	DeviceID    string `json:"id"`        //设备ID
	RtmpURL1    string `json:"rtmp1"`     //设备rtmp URL1
	RtmpURL2    string `json:"rtmp2"`     //设备rtmp URL2
	Destription string `json:"des"`       //设备描述
	Thumbnail   string `json:"thub"`      //设备缩略图
	Cost        int    `json:"cost"`      //设备游戏花费
	DesImg      string `json:"desImgUrl"` //游戏内物品介绍图片URL
	Play        int    `json:"play"`      //游戏次数
	Success     int    `json:"success"`   //成功次数
	LeftCount   int    `json:"left"`      //剩余数量
	Force       int    `json:"force"`     //娃娃机抓力(多少次抓住一次！！)
	Exchange    int    `json:"exchange"`  //兑换娃娃币数量
}

//SCAllDevices 所有娃娃机设备信息
type SCAllDevices struct {
	AllDevices []SCDeviceInfo `json:"devices"` //所有注册过的设备
}

//SCDeviceRegRet 娃娃机注册消息返回
type SCDeviceRegRet struct {
	Success     bool   `json:"ok"`   //是否成功
	UserID      int64  `json:"user"` //用户ID
	Destription string `json:"des"`  //成功或者失败描述
}

//SCDeviceCoin 娃娃机投币信息
type SCDeviceCoin struct {
	DeviceID  string `json:"did"` //设备ID
	BigRoomID string `json:"rid"` //bigroom id
}

//SCCoinRet 娃娃机投币结果信息
type SCCoinRet struct {
	Success     bool   `json:"state"` //投币是否成功
	Gold        int64  `json:"gold"`  //成功后剩余金币
	Destription string `json:"des"`   //投币结果描述
}

//SCDeviceAction 娃娃机操作信息
type SCDeviceAction struct {
	DeviceID string `json:"id"`     //设备ID
	Action   int    `json:"action"` //操作信息
}

//CSQueryRoomInfo 客户端查询相应房间信息
type CSQueryRoomInfo struct {
	GameID int `json:"id"` //房间ID，或者说gameID
}

//SCActiveInfo 活动信息
type SCActiveInfo struct {
	ActiveID int    `json:"id"`  //活动ID
	ImgURL   string `json:"url"` //活动图片
}

//SCActivesInfo 返回相应活动信息
type SCActivesInfo struct {
	Actives []SCActiveInfo `json:"actives"` //活动信息
}

//SCTableInfoRet 查询table信息返回
type SCTableInfoRet struct {
	Device     SCDeviceInfo `json:"dev"` //设备信息
	TableCount int          `json:"num"` //该桌玩家个数
	BigRoomID  string       `json:"id"`  //bigroomid
}

//SCRoomInfoRet 查询房间信息返回
type SCRoomInfoRet struct {
	Success bool             `json:"ok"`     //查询是否成功
	Devices []SCTableInfoRet `json:"tables"` //talbe信息
}

//SCRoomLast2Players 加入房间成功后返回当前房间最后两人的头像
type SCRoomLast2Players struct {
	Users [2]string `json:"ids"`   //用户账号
	Heads [2]string `json:"heads"` //用户头像
}

//CSRoomChat 房间聊天
type CSRoomChat struct {
	DeviceID string `json:"id"`   //设备ID（对应房间）
	ChatInfo string `json:"chat"` //聊天信息
}

//SCRoomChat 广播房间内聊天
type SCRoomChat struct {
	UserNick string `json:"user"` //发言玩家昵称
	ChatInfo string `json:"chat"` //聊天信息
}

//SCSuccessRecord 成功抓娃娃记录
type SCSuccessRecord struct {
	UserID   string `json:"id"`       //玩家Accounts
	UserNick string `json:"user"`     //成功玩家昵称
	UserHead string `json:"head"`     //玩家头像URL
	Date     string `json:"date"`     //成功抓取娃娃的时间
	Video    string `json:"video"`    //录像地址
	DeviceID string `json:"deviceID"` //娃娃ID描述
}

//CSSuccessList 查询成功抓娃娃记录
type CSSuccessList struct {
	DeviceID string `json:"did"` //娃娃机ID
}

//SCSuccessList 成功抓娃娃记录列表
type SCSuccessList struct {
	Records []SCSuccessRecord `json:"records"` //成功记录
}

//SCWawaList 娃娃列表
type SCWawaList struct {
	Records  []SCSuccessRecord `json:"records"`  //成功记录
	Name     []string          `json:"name"`     //娃娃名称
	Thub     []string          `json:"thub"`     //娃娃缩略图
	State    []int             `json:"state"`    //发货状态
	Exchange []int             `json:"exchange"` //兑换价格状态
}

//CSWaWaExchange 娃娃兑换娃娃币
type CSWaWaExchange struct {
	WaWaID int `json:"idx"` //库存娃娃索引
}

//SCWaWaExchangeRet 娃娃兑换娃娃币
type SCWaWaExchangeRet struct {
	List SCWawaList `json:"list"` //当前娃娃列表
	Gold int        `json:"gold"` //当前金币数
}

//CSAskDelivery 发货请求
type CSAskDelivery struct {
	DeliveryIDs string        `json:"ids"`  //请求发货的索引
	Address     SCAddressInfo `json:"addr"` //请求发货的信息
}

//SCOneRecord 一条货物记录
type SCOneRecord struct {
	Records SCSuccessRecord `json:"records"` //成功记录（中奖时间、中奖用户ID，昵称，物品描述）
	//Name    []string          `json:"name"`    //娃娃名称（）
	Address SCAddressInfo `json:"addr"` //请求发货的地址信息
}

//SCBGRecordSum 后台返回货物相关信息
type SCBGRecordSum struct {
	Records   []SCOneRecord `json:"records"` //货物记录
	CurPage   int           `json:"cur"`     //当前页数
	TotalPage int           `json:"total"`   //总页数
}

//SCSuccessRank 成功抓娃娃记录列表
type SCSuccessRank struct {
	SelfRank     int      `json:"rank"`   //查询用户排名
	SelfScore    int      `json:"score"`  //查询用户成功次数
	UserNicks    []string `json:"users"`  //用户昵称
	SuccessCount []string `json:"counts"` //用户成功次数
	UserHead     []string `json:"head"`   //用户头像
}

//SCMainInfo 邮箱信息
type SCMainInfo struct {
	ID        string `json:"mailID"` //邮件ID
	SystemMsg bool   `json:"sys"`    //是否系统邮件
	Title     string `json:"title"`  //邮件标题
	MailDes   string `json:"mail"`   //邮件信息
	Read      bool   `json:"read"`   //邮件状态 false：未读 true：已读
	Reward    int    `json:"reward"` //奖励数量
	Date      string `json:"date"`   //时间
}

//SCAllMainInfo 客户端查询邮箱信息
type SCAllMainInfo struct {
	Mails []SCMainInfo `json:"mail"` //用户账号
}

//SCShopItemInfo 商品信息
type SCShopItemInfo struct {
	ID       string `json:"shopID"` //商品ID
	Title    string `json:"title"`  //商品标题
	ShopDes  string `json:"des"`    //商品描述信息
	ExtraDes string `json:"extra"`  //额外描述信息
	Type     int    `json:"type"`   //商品类型 （0：直接兑换金币，1： 周卡 2：月卡）
	ExtraGot int    `json:"ext"`    //周卡月卡时每天返还金币数量
	Cost     int    `json:"cost"`   //商品花费
	Gold     int    `json:"gold"`   //获得金币数量
	ImageURL string `json:"url"`    //商品图片URL
}

//SCAllShopItems 商品信息
type SCAllShopItems struct {
	AllItems []SCShopItemInfo `json:"items"` //所有商品信息
}

//SCShopInfos 获取商品信息返回
type SCShopInfos struct {
	AllItems  SCAllShopItems `json:"items"`  //所有商品信息
	CardState int            `json:"cards"`  //周卡，月卡信息 0表示未购买周卡月卡， 1 ： 周卡 ，2： 月卡， 3：周卡月卡都买了
	GetState  int            `json:"states"` //当天领取周卡月卡信息 0表示未领取周卡月卡， 1 ： 已领取周卡 ，2： 已领取月卡， 3：周卡月卡都领取了
}

//CSBuyItem 客户端购买商品信息
type CSBuyItem struct {
	ShopID string `json:"shopID"` //商品ID
}

//SCPrePayID 微信H5支付prepayID (即跳转url)
type SCPrePayID struct {
	PrePayID    string `json:"id"`    //微信H5支付prepayID
	TradeNumber string `json:"trade"` //商品ID
}

//CSBuyDone 客户端购买商品信息返回
type CSBuyDone struct {
	TradeNumber string `json:"trade"` //商品ID
}

//SCBuyResult 服务端购买商品结果返回
type SCBuyResult struct {
	State       int    `json:"state"` //购买是否成功
	Description string `json:"des"`   //描述
	NowGold     int    `json:"gold"`  //成功后现有娃娃币
}

//SCTradeInfo 订单信息
type SCTradeInfo struct {
	UserID      int    `json:"id"`     //用户ID
	UserNick    string `json:"nick"`   //用户昵称
	Type        string `json:"type"`   //充值类型
	ItemID      int    `json:"itemID"` //充值商品ID
	Cost        int    `json:"cost"`   //花费人民币数目
	Gold        int    `json:"gold"`   //得到娃娃币数目
	Description string `json:"des"`    //充值描述
	Date        string `json:"date"`   //充值时间
	State       int    `json:"state"`  //充值状态
}

//SCAllTradeInfos 订单信息
type SCAllTradeInfos struct {
	Trades    []SCTradeInfo `json:"trades"` //订单信息
	CurPage   int           `json:"cur"`    //当前页数
	TotalPage int           `json:"total"`  //总页数
}

//SCCheckInCount 连续签到次数
type SCCheckInCount struct {
	Today bool `json:"today"` //今天是否已经签到
	Count int  `json:"sign"`  //连续签到次数
	Gold  int  `json:"gold"`  //当前金币数
}

//SCAddressInfo 地址信息
type SCAddressInfo struct {
	Name string `json:"name"` //联系人
	Tel  string `json:"tel"`  //联系人电话
	Area string `json:"area"` //地区信息
	Addr string `json:"addr"` //详细地址
}

//SCAddresss 所有地址信息
type SCAddresss struct {
	Addresss []SCAddressInfo `json:"addrs"` //所有地址信息
}

//SCInviteUserInfo 绑定用户信息
type SCInviteUserInfo struct {
	UserAC   string `json:"ac"`    //玩家Accounts
	UserNick string `json:"nick"`  //玩家昵称
	UserHead string `json:"head"`  //玩家头像
	Money    int    `json:"money"` //奖励金
}

//SCInviteInfo 绑定信息
type SCInviteInfo struct {
	Code       string             `json:"code"`    //自己的邀请码
	LeftReward int                `json:"left"`    //奖金余额
	BindedUser string             `json:"binded"`  //已绑定的用户
	Binders    []SCInviteUserInfo `json:"binders"` //绑定用户信息
}

//CSBindCode 绑定邀请码信息
type CSBindCode struct {
	Code string `json:"code"` //邀请码
}

//SCBindRet 绑定邀请码信息
type SCBindRet struct {
	State       bool   `json:"state"` //绑定是否成功
	Description string `json:"des"`   //描述
}

//SCBGUserInfo 服务器用户信息后台
type SCBGUserInfo struct {
	ID           int64  `json:"id"`   //用户ID
	NickName     string `json:"nick"` //昵称
	Gender       bool   `json:"sex"`  //性别
	Gold         int64  `json:"gold"` //金币
	RegisterDate string `json:"date"` //注册日期
}

//SCBGUserInfoSum 服务器用户信息后台
type SCBGUserInfoSum struct {
	Users     []SCBGUserInfo `json:"users"` //用户信息
	CurPage   int            `json:"cur"`   //当前页数
	TotalPage int            `json:"total"` //总页数
}

//CSShareDone 客户端分享成功
type CSShareDone struct {
	SharePlace string `json:"place"` //分享大厅还是游戏内
}
