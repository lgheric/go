package main

import (
	"git.apache.org/thrift.git/lib/go/thrift"
	"github.com/gen-go/Services/AuctionService"
	"log"
	"fmt"
)

func biddingThriftClient(dealerId int64, orderId int64, basePrice float64, addPrice float64)(result bool) {

	// 服务运行方式
	http, err := thrift.NewTHttpPostClient(conf.Thrift.Biddinghost)
	if err != nil {
		log.Printf("Error thrift 18 opening http failed: %v \n", err)
	} else {
		// 数据传输方式
		transport := thrift.NewTBufferedTransport(http, 24*1024*1024)
		// 数据格式
		var protocol thrift.TProtocol = thrift.NewTBinaryProtocolTransport(transport)

		protocol = thrift.NewTMultiplexedProtocol(protocol, "AuctionService")
		client := AuctionService.NewAuctionServiceClientProtocol(transport, protocol, protocol)
		err := transport.Open();
		defer client.Transport.Close()
		if  err != nil {
			log.Printf("Error thrift 25 opening transport failed: %v \n", err)
		} else {
			result, err = client.IntelBidding(dealerId, orderId, basePrice, addPrice)
			if err != nil {
				log.Println(err)
			}
			trace(fmt.Sprintf("dealer : %d | order : %d | addPrice %f | bidding : %v", dealerId, orderId , addPrice, result))
		}

	}
	return

}


