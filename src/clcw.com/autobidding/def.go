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
	Host         string
	Port         string
	Pswd         string
	Database     string
	OrderInfo    string
	Orderlist    string
	Dealerlist   string
	Orderbidding string
	DealerInfo   string
	Timeout      int
}

type logConf struct {
	File string
	Debug int
}

type thriftConf struct {
	Biddinghost string
}

type biddingConf struct {
	AddPrice int
}

type config struct {
	Mysql   mysqlConf
	Redis   redisConf
	Log     logConf
	Thrift  thriftConf
	Bidding biddingConf
}

type order struct {
	OrderId          int64   `json:"order_id"`
	BiddingStartTime string  `json:"bidding_start_time"`
	BidStartPrice    float64 `json:"bid_start_price"`
}

type orderBidding struct {
	DealerId    int64   `json:"dealer_id"`
	OrderId     int64   `json:"order_id"`
	BeforePrice float64 `json:"before_price"`
	AppendPrice float64 `json:"append_price"`
	AfterPrice  float64 `json:"after_price"`
	BiddingType int32   `json:"bidding_type"`
	CreateTime  int64   `json:"create_time"`
}

type orderDealer struct {
	DealerId   int64   `json:"dealer_id"`
	MaxPrice   float64 `json:"max_price"`
	BidCount   int64   `json:"bid_count"`
	CreateTime int64   `json:"create_time"`
}
