package main

//toml type
type mysqlConf struct {
	Host     string
	Port     string
	User     string
	Pswd     string
	Charset  string
	Database string
}

type redisConf struct {
	Host       string
	Port       string
	Pswd       string
	Startq     string
	Endq       string
	Scene      string
	Dealerlock string
	Orderlock  string
	Cardlock   string
	Scenelock   string
	Timeout    int
	AutoOrderLs string
	AutoOrderIf string
	AutoOrderBi string
	AutoDealerLs string
	AutoDealerIf string
}

type logConf struct {
	File string
}

type pprofConf struct {
	Host string
	Port string
}

type thriftConf struct {
	Host string
	Port string
}

type feeConf struct {
	Percentage float64
	Minfee     int
	Maxfee     int
}

type config struct {
	Mysql mysqlConf
	Redis redisConf
	Log   logConf
	Fee   feeConf
	Pprof	pprofConf
	Thrift	thriftConf
}

type order struct {
	OrderId             int     // 拍单id
	OrderNo             string  // 拍单号
	SceneId             int     // 拍场id
	CarId               int     // 车辆编号
	Status              int     // 拍单状态(-1 删除,1审核中,2审核驳回,3投标中,301等待竞标,4竟标中,5待确认,7已确认,8已签约,9待过户,10过户中,11过户完成,12拍单失败,13拍单成功
	Rank                int     // 定时拍排名
	IsTimingOrder       bool    // 是否为定时单
	BidStartTime        string  // 投标开始时间(即上拍审核通过的时间）
	BiddingStartTime    string  // 竞拍开始时间(即场次开始时间）
	BiddingEndTime      string  // 竞拍结束时间(后台PHP进程更新）
	EstElapsedTime      int     // 预计拍卖耗时
	ActElapsedTime      int     // 实际拍卖耗时
	BiddingBestDealerId int     // 竞标出价最高的车商ID
	BidBestDealerId     int     // 投标出价最高的车商ID
	BiddingBestPrice    float64 // 竞标阶段最高价(第一名)
	BidBestPrice        float64 // 投标阶段最高价
	FirstMoney          float64 // 应付首款
	SuccessPrice        float64 // 成交价格
	SuccessDealerId     int // 第一名
	CompanySubsidies    float64 // 公司补贴
	TailMoney           float64 // 应付尾款
}

type car struct {
	CarId          int
	CarNo          string
	Sid		int
	Sno		string
	OwnerId        int
	IsDealerBreach int // 是否车商违约(0否1是)
	CarSource      int  // 车辆来源(1,4S店 2,个人)
	PayStatus      int  // 付款状态：-1 付款关闭 1、待付首款 2、已付首款 3、待付尾款 4、已付尾款
	DeliveryMode   int  // 交付模式(1先付款后验车,2先验车后付款)
	ThreeInOne     int  // 三证合一(1,是 2,否)
	LocationArea   int  // 车辆所在地
	IsSelfReceive	int //是否自收车辆
	SelfReceiveDealerId int //自收车辆车商ID
}

type dealerBailLog struct {
	blId       int
	dealerId   int
	occurMoney float64
}

type StartOrderQueue struct {
	OrderID          int    `json:"order_id"`
	SceneID          int    `json:"scene_id"`
	BiddingStartTime string `json:"bidding_start_time"`
	BiddingEndTime   string `json:"bidding_end_time"`
	EstElapsedTime   int    `json:"est_elapsed_time"`
	ActElapsedTime   int    `json:"act_elapsed_time"`
	Rank             int    `json:"rank"`
	IsTimingOrder    int    `json:"is_timing_order"`
}

type EndOrderQueue struct {
	OrderID          int    `json:"order_id"`
	SceneID          int    `json:"scene_id"`
	BiddingStartTime string `json:"bidding_start_time"`
	BiddingEndTime   string `json:"bidding_end_time"`
	EstElapsedTime   int    `json:"est_elapsed_time"`
	ActElapsedTime   int    `json:"act_elapsed_time"`
	Rank             int    `json:"rank"`
	IsTimingOrder    int    `json:"is_timing_order"`
}
