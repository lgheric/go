package main

import (
	//"github.com/gen-go/auction"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"gen-go/Services/BailService"
	"log"
	"time"
)

func thriftClient() {
	//startTime := currentTimeMillis()
	url := "http://"+conf.Thrift.Host+":"+conf.Thrift.Port+"/Bail/run"

	transport, err := thrift.NewTHttpPostClient(url)
	if err != nil {
		log.Fatalf("Error thrift 18 opening http failed: %v", err)
	}

	var protocol thrift.TProtocol = thrift.NewTBinaryProtocolTransport(transport)
	protocol = thrift.NewTMultiplexedProtocol(protocol, "BailService")
	client := BailService.NewBailServiceClientProtocol(transport, protocol, protocol)
	if err := transport.Open(); err != nil {
		log.Fatalf("Error thrift 25 opening transport failed: %v", err)
	}
	defer client.Transport.Close()

	//获取车商保证金信息(可用保证金、冻结保证金、扣除保证金)
	var (
		dealerID int64 = 242
		orderId int64 = 173
	)
	//bk, err := client.GetDealerBail(dealerID)
	//if err != nil {
	//	fmt.Println(err)
	//}
	//fmt.Println(bk)

	//解冻车商保证金
	bail ,err := client.UnfreezeBail(dealerID,orderId)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(bail)


	//endTime := currentTimeMillis()
	//fmt.Println("Program exit. time->", endTime, startTime, (endTime - startTime))

}

func currentTimeMillis() int64 {
	return time.Now().UnixNano() / 1000000
}
