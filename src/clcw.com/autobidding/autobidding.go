package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

/*
 * 智能拍单监听
 */
func orderMonitor() {

	key := conf.Redis.Orderlist
	for {
		orderIds := smembers(key)
		trace(fmt.Sprintf("all order : %v", orderIds))
		if len(orderIds) > 0 {
			orderDistribute(orderIds)
		}
		time.Sleep(time.Second)
	}

}

/*
 * 智能拍单分发
 */
func orderDistribute(orderIds []int) {

	for _, val := range orderIds {
		go orderHandle(int64(val))
	}

}

/*
 * 单个智能拍单处理
 */
func orderHandle(orderId int64) {

	startTime := currentTimeMillis()

	key := conf.Redis.Orderbidding + fmt.Sprintf("%d", orderId)
	str := redisGet(key)
	if len(str) > 0 {
		// 有出价信息
		bidding := &orderBidding{}
		err := json.Unmarshal([]byte(str), &bidding)
		if err != nil {
			log.Printf("autobidding 42 json %s failed: %v \n", key, err)
		} else {
			if bidding.BiddingType == 1 {
				// 上一次是人出价  则机器立刻出价
				getBidDealer(orderId, bidding.DealerId, bidding.AfterPrice)

			} else if bidding.BiddingType == 2 {
				// 上一次是机器出价  则等20s 机器立刻出价
				now := time.Now().Unix()
				diff := now - bidding.CreateTime
				// 超过上拍时间 20s 开始出价
				if diff >= 20 {
					getBidDealer(orderId, bidding.DealerId, bidding.AfterPrice)
				}
			}
		}
	} else {
		// 没有出价信息 拿到订单开始时间。
		orderKey := conf.Redis.OrderInfo + fmt.Sprintf("%d", orderId)
		orderStr := redisGet(orderKey)
		if len(orderStr) > 0 {
			order := &order{}
			err := json.Unmarshal([]byte(orderStr), &order)
			if err != nil {
				log.Printf("autobidding 68 json %s failed: %v \n", orderKey, err)
			} else {
				now := time.Now().Unix()
				startTime, _ := formatTime(order.BiddingStartTime)
				diff := now - startTime
				// 超过上拍时间 20s 开始出价
				if diff >= 20 {
					getBidDealer(orderId, 0, order.BidStartPrice)
				}
			}
		} else {
			log.Printf("redis %d order info is empty \n", orderId)
		}

	}

	endTime := currentTimeMillis()
	trace(fmt.Sprintf("start: %d end: %d time-> %d \n", endTime, startTime, (endTime - startTime)))

}

/*
 * 出价规则 匹配下一个车商然后出价
 */
func getBidDealer(orderId int64, biddingDealerID int64, basePrice float64) {

	dealerListkey := conf.Redis.Dealerlist + fmt.Sprintf("%d", orderId)
	list := zrange(dealerListkey, 0, -1)
	trace(fmt.Sprintf("order : %d all dealer : %v ", orderId, list))
	if len(list) > 0 {
		for _, dealerId := range list {
			trace(fmt.Sprintf("now bidding dealer : %d ", dealerId))
			if dealerId > 0 {
				dealerKey := conf.Redis.DealerInfo + fmt.Sprintf("%d_%d", orderId, dealerId)
				dealer := &orderDealer{}
				dealerStr := redisGet(dealerKey)
				err := json.Unmarshal([]byte(dealerStr), &dealer)
				if err != nil {
					log.Printf("autobidding 95 json %s failed: %v \n", dealerKey, err)
				} else {
					trace(fmt.Sprintf("dealerId : %d  biddingCount : %d ", dealerId, dealer.BidCount))
					if biddingDealerID > 0 {
						// 有上次出价
						// 判断上次是不是自己 不是则出价
						if biddingDealerID != dealer.DealerId {
							addPrice := float64(conf.Bidding.AddPrice)
							biddingPrice := addPrice + basePrice
							// 判断是否超过车商设置的最大出价
							if biddingPrice <= dealer.MaxPrice {
								result := biddingThriftClient(dealer.DealerId, orderId, basePrice, addPrice)
								if result {
									return
								} else {
									continue
								}
							}
						}
					} else {
						// 无出价
						addPrice := float64(conf.Bidding.AddPrice)
						biddingPrice := addPrice + basePrice
						// 判断是否超过车商设置的最大出价
						if biddingPrice <= dealer.MaxPrice {
							result := biddingThriftClient(dealer.DealerId, orderId, basePrice, addPrice)
							if result {
								return
							} else {
								continue
							}

						}
					}
				}
			}
		}
	}

}
