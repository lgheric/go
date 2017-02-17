package main

import (
	//"github.com/gen-go/auction"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"github.com/gen-go/bail"
	"log"
	"time"
)

func thriftClient() {
	startTime := currentTimeMillis()
	url := "http://"+conf.Thrift.Host+":"+conf.Thrift.Port+"/Bail/run"

	transport, err := thrift.NewTHttpPostClient(url)
	if err != nil {
		log.Fatalf("Error thrift 18 opening http failed: %v", err)
	}

	var protocol thrift.TProtocol = thrift.NewTBinaryProtocolTransport(transport)
	protocol = thrift.NewTMultiplexedProtocol(protocol, "BailService")
	client := bail.NewBailServiceClientProtocol(transport, protocol, protocol)
	if err := transport.Open(); err != nil {
		log.Fatalf("Error thrift 25 opening transport failed: %v", err)
	}
	defer client.Transport.Close()

	var dealerID int64 = 5
	bk, err := client.GetDealerBail(dealerID)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(bk)

	endTime := currentTimeMillis()
	fmt.Println("Program exit. time->", endTime, startTime, (endTime - startTime))

}

func currentTimeMillis() int64 {
	return time.Now().UnixNano() / 1000000
}
