package main

import (
	"encoding/json"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"log"
	"time"

	"database/sql"
)

// 处理开始时间的队列
func orderStartService() {

	conn := pool.Get()
	defer conn.Close()

	for {
		now := time.Now().Unix()
		// now = 123456
		queueName := conf.Redis.Startq + fmt.Sprintf("%d", now)
		keyExists, err := redis.Bool(conn.Do("EXISTS", queueName))
		if err != nil {
			log.Fatalf("%s queue exists failed: %s ", queueName, err)
		}
		if keyExists {
			// 这里速度太快会全部取掉，就会异常（ redigo: nil returned）。忽略掉这个异常
			res, _ := redis.String(conn.Do("RPOP", queueName))
			fmt.Printf("%s queue pop %s \n", queueName, res)
			go orderStartHandle(res)
		}
	}

}
//开始处理单个开始拍单
func orderStartHandle(res string) {
	order := &StartOrderQueue{}
	err := json.Unmarshal([]byte(res), &order)
	if err != nil {
		log.Fatalf("Json: %s 解析失败:%v",res, err)
	}

	if order.SceneID == 0 || order.OrderID == 0 {
		log.Printf("order : %d, scene : %d 无效", order.OrderID, order.SceneID)
		return
	}

	conn := pool.Get()
	defer conn.Close()

	//获取场redis key
	count, err := redis.Int(conn.Do("SCARD", conf.Redis.Scene+fmt.Sprintf("%d", order.SceneID)))
	if err != nil {
		log.Fatalf("Redis Command:SCARD %s %d 发生致命错误：%v \n", conf.Redis.Scene,order.SceneID,err)
	}

	fmt.Printf("scene_id : %d scard : %d \n", order.SceneID, count)
	if count <= 0 {
		oids := getOrderList(order.SceneID)
		if len(oids) > 0 {
			sceneStart(order.SceneID)
			orderWaitBidding(oids, order.SceneID)
			sceneSaddOrder(oids, order.SceneID)
		} else {
			fmt.Printf("场%d 没有待投标的订单", order.SceneID)
			return
		}
	}

	orderStart(order.OrderID)

	ov, _ := json.Marshal(order)
	stime, _ := formatTime(order.BiddingStartTime)
	etime := stime + int64(order.EstElapsedTime)

	queueName := conf.Redis.Endq + fmt.Sprintf("%d", etime)
	lpush(queueName, ov)
	fmt.Printf("StartOrderQueue %s : scene_id : %d | order_id : %d \n", queueName, order.SceneID, order.OrderID)
}

// 处理结束时间的队列
func orderEndService() {

	conn := pool.Get()
	defer conn.Close()

	for {
		now := time.Now().Unix()
		// now = 1477561980
		queueName := conf.Redis.Endq + fmt.Sprintf("%d", now)
		keyExists, err := redis.Bool(conn.Do("EXISTS", queueName))
		if err != nil {
			log.Fatalf("%s queue exists failed: %s ", queueName, err)
		}
		if keyExists {
			// 这里速度太快会全部取掉，就会异常（ redigo: nil returned）。忽略掉这个异常
			res, _ := redis.String(conn.Do("RPOP", queueName))
			fmt.Printf("%s queue pop %s \n", queueName, res)
			go orderEndHandle(res)
		}
	}

}


//开始处理单个结束拍单
func orderEndHandle(res string) {
	eod := &EndOrderQueue{}
	err := json.Unmarshal([]byte(res), &eod)
	if err != nil {
		log.Fatalf("Json %s 解析 failed: %v \n", res, err)
	}

	order := getOrder(eod.OrderID)

	if order.OrderId == 0 {
		log.Printf("order : %d, scene : %d 无效", order.OrderId, order.SceneId)
		return
	}

	if order.Status != 4 {
		return
	}

	ftime, _ := formatTime(order.BiddingStartTime)
	endTime := ftime + int64(order.ActElapsedTime)
	now := time.Now().Unix()
	if endTime > now {
		ov, _ := json.Marshal(eod)
		queueName := conf.Redis.Endq + fmt.Sprintf("%d", endTime)
		lpush(queueName, ov)
		fmt.Printf("EndOrderQueue %s : scene_id : %d | order_id : %d \n", queueName, order.SceneId, order.OrderId)

	} else {
		if order.CarId > 0 {
			orderEnd(order)
			sceneSremOrder(order.OrderId, order.SceneId)
			key := conf.Redis.Scene + fmt.Sprintf("%d", order.SceneId)
			keyExists, _ := exists(key)
			if !keyExists {
				sceneEnd(order.SceneId)
			}
		}
	}

}
// 处理指定时间段开拍的拍单
func orderStartServiceByTime(startTime int64) {

	conn := pool.Get()
	defer conn.Close()
	for i := startTime; startTime < 2147443200; i++ {
		queueName := conf.Redis.Startq + fmt.Sprintf("%d", i)
		keyExists, err := redis.Bool(conn.Do("EXISTS", queueName))
		if err != nil {
			log.Fatalf("%s queue exists failed: %s ", queueName, err)
		}
		if keyExists {
			// 这里速度太快会全部取掉，就会异常（ redigo: nil returned）。忽略掉这个异常
			res, _ := redis.String(conn.Do("RPOP", queueName))
			fmt.Printf("%s queue pop %s \n", queueName, res)
			//go orderStartHandle(res)
		}
	}
}

// 处理指定时间段结束的拍单
func orderEndServiceByTime(startTime int64) {

	conn := pool.Get()
	defer conn.Close()

	for i := startTime; startTime < 2147443200; i++ {
		queueName := conf.Redis.Endq + fmt.Sprintf("%d", i)
		keyExists, err := redis.Bool(conn.Do("EXISTS", queueName))
		if err != nil {
			log.Fatalf("%s queue exists failed: %s ", queueName, err)
		}
		if keyExists {
			// 这里速度太快会全部取掉，就会异常（ redigo: nil returned）。忽略掉这个异常
			res, _ := redis.String(conn.Do("RPOP", queueName))
			fmt.Printf("%s queue pop %s \n", queueName, res)
			go orderEndHandle(res)
		}
	}
}



//处理订单
func orderStart(oid int) {
	var (
		price    float64
		dealerID int
	)

	stmt := "SELECT price,dealer_id FROM au_bid_log a INNER JOIN (SELECT MAX(bid_id) bid_id FROM au_bid_log WHERE order_id = ? GROUP BY dealer_id) b ON a.`bid_id` = b.bid_id ORDER BY price DESC,createtime LIMIT 1"
	rows := db.QueryRow(stmt, oid)
	err := rows.Scan(&price, &dealerID)

	// 可能无人投标出价
	//if err != nil {
	//	log.Fatalf("in order line 218 mysql fetch result failed: %v ", err)
	//}

	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("order start mysql transaction begin failed: %v ", err)
	}
	defer tx.Rollback()

	stm, err := tx.Prepare("UPDATE `au_order` SET bid_best_price = ?, bid_best_dealer_id = ?, `bidding_start_time` = now(), `status` = 4 WHERE `order_id` = ?")
	stm.Exec(price, dealerID, oid)

	//rowCnt, err := result.RowsAffected()
	//fmt.Println(rowCnt)
	//if err != nil {
	//	log.Fatalf(err)
	//}

	if dealerID > 0 {
		paimaiRefund(oid, dealerID)
		fmt.Printf("订单 %d 解冻投标阶段的保证金 \n", oid)
	} else {
		fmt.Printf("订单：%d 无人投标 \n", oid)
	}

	order := getOrder(oid)

	stmtl, _ := tx.Prepare("INSERT INTO `au_order_trace_log_list`(`order_id`, `car_id`, `emp_name`, `action_no`, `action_name`, `createtime`) VALUES (?, ?, ?, ?, ?, ?)")
	stmtl.Exec(oid, order.CarId, "--", 1007, "开始竞拍", time.Now().UnixNano())

	car := getCar(order.CarId)

	stmtc, _ := tx.Prepare("INSERT INTO `au_car_trace_log_list`(`owner_id`, `car_id`, `emp_name`, `action_no`, `action_name`, `createtime`) VALUES (?, ?, ?, ?, ?, ?)")
	stmtc.Exec(car.OwnerId, car.CarId, "--", 1013, "开始竞拍", time.Now().UnixNano())

	errc := tx.Commit()
	if errc != nil {
		log.Fatalf("order start mysql transaction commit failed: %v ", errc)
	}

	if order.IsTimingOrder {
		fmt.Printf("%d : 定时拍单%s竞拍中。。。。。。。。\n", order.SceneId, order.OrderNo)
	} else {
		fmt.Printf("%d : 顺序拍单%s竞拍中。。。。。。。。\n", order.SceneId, order.OrderNo)
	}

}

//处理订单
func orderEnd(od order) {

	fmt.Printf("开始结束订单: %d \n", od.OrderId)

	key := conf.Redis.Orderlock + fmt.Sprintf("%d", od.OrderId)
	keyBool, _ := exists(key)
	if !keyBool {
		lockBool := redisLock(key)
		if lockBool {
			tx, err := db.Begin()
			if err != nil {
				log.Fatalf("order end mysql transaction begin failed: %v ", err)
			}
			defer tx.Rollback()

			stmt, _ := tx.Prepare("UPDATE `au_order` SET `status` = 5, `last_time` = NOW(), `bidding_end_time` = NOW() WHERE `order_id` = ?")
			stmt.Exec(od.OrderId)

			if od.BidBestPrice >= od.BiddingBestPrice {
				stmtd, _ := tx.Prepare("INSERT INTO `au_dealer_auctioned` SET `dealer_id` = ?, `order_id` = ?, `createtime` = NOW(), `trade_price` = ?")
				stmtd.Exec(od.BidBestDealerId, od.OrderId, od.BidBestPrice)

				//保证金
				paimaiRefund(od.OrderId, od.BidBestDealerId)

				//更新交易服务费
				commision := getCommision(od.BidBestPrice)
				stmt, _ := tx.Prepare("UPDATE `au_order` SET `success_price` = ?, `success_dealer_id` = ?, `commision` = ? WHERE `order_id` = ?")
				stmt.Exec(od.BidBestPrice, od.BidBestDealerId, commision, od.OrderId)
			} else {
				stmtd, _ := tx.Prepare("INSERT INTO `au_dealer_auctioned` SET `dealer_id` = ?, `order_id` = ?, `createtime` = NOW(), `trade_price` = ?")
				stmtd.Exec(od.BiddingBestDealerId, od.OrderId, od.BiddingBestPrice)

				//保证金
				paimaiRefund(od.OrderId, od.BiddingBestDealerId)

				//更新交易服务费
				commision := getCommision(od.BiddingBestPrice)
				stmt, _ := tx.Prepare("UPDATE `au_order` SET `success_price` = ?, `success_dealer_id` = ?, `commision` = ? WHERE `order_id` = ?")
				stmt.Exec(od.BiddingBestPrice, od.BiddingBestDealerId, commision, od.OrderId)
			}
			car := getCar(od.CarId)

			//更新拍单进度状态
			updateTraceLog(od,car,tx)

			//更新车源为待确认
			updateCarSource(car,tx)

			//处理违约重拍 -- 到平台确认
			breachRedoPlatform(od,car,tx)

			//处理自收重拍
			SelfReceiveRedo(od.OrderId,car,tx)

			errc := tx.Commit()
			if errc != nil {
				log.Fatalf("order start mysql transaction commit failed: %v ", errc)
			}

			if od.IsTimingOrder {
				fmt.Printf("%d : 定时拍单%s竞拍结束 \n", od.SceneId, od.OrderNo)
			} else {
				fmt.Printf("%d : 顺序拍单%s竞拍结束 \n", od.SceneId, od.OrderNo)
			}
			redisUnLock(key)
		}

	} else {
		time.Sleep(time.Second)
		orderEnd(od)
	}

}
//更新拍单进度状态
func updateTraceLog(od order,car car,tx *sql.Tx){

	stmtl, _ := tx.Prepare("INSERT INTO `au_order_trace_log_list`(`order_id`, `car_id`, `emp_name`, `action_no`, `action_name`, `createtime`) VALUES (?, ?, ?, ?, ?, ?)")
	stmtl.Exec(od.OrderId, od.CarId, "--", 1008, "竞拍结束", time.Now().UnixNano())

	stmtc, _ := tx.Prepare("INSERT INTO `au_car_trace_log_list`(`owner_id`, `car_id`, `emp_name`, `action_no`, `action_name`, `createtime`) VALUES (?, ?, ?, ?, ?, ?)")
	stmtc.Exec(car.OwnerId, car.CarId, "--", 1014, "竞拍结束", time.Now().UnixNano())

}

//更新车源状态
func updateCarSource(car car,tx *sql.Tx) {

	stmt := "SELECT `status` FROM `au_car_source` WHERE `sid` = ? LIMIT 1"
	rows := db.QueryRow(stmt, car.Sid)

	var status int
	err := rows.Scan(&status)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Println("找不到车源sid:",car.Sid)
		} else {
			log.Printf("select status from  au_car_source where sid= %d 543 mysql fetch result failed: %v ", car.Sid,err)
		}
	}

	if status == 200 {
		stmt, _ := tx.Prepare("UPDATE `au_car_source` SET `status` = 300 WHERE `sid` = ?")
		stmt.Exec(car.Sid)

		fmt.Printf("更新车源 %d 状态为300 待确认\n", car.Sid)
	}

}

//处理违约重拍 -- 到平台确认
func breachRedoPlatform(order order,car car,tx *sql.Tx){

	if car.IsDealerBreach == 1  {
		log.Println(order.OrderId,"-",order.OrderNo,"违约重拍处理开始。。")

		//获取上一个拍单首付款信息
		var firtMoney float64
		stmt := "SELECT first_money FROM `au_order` WHERE `car_id`=? ORDER BY order_id DESC limit 1,1"
		row := db.QueryRow(stmt,car.CarId)
		err := row.Scan(&firtMoney)
		if err != nil {
			if err == sql.ErrNoRows {
				log.Println("根据car_id未找到之前的拍单:",order.CarId)
			}else{
				log.Fatalf("order.go line 380 sql query row. sql: %d err: %v",order.OrderId,err)
			}
		}

		stmt1, _ := tx.Prepare("UPDATE au_order SET first_money=?,confirm_type = ? WHERE order_id = ?")
		stmt1.Exec(1,firtMoney,order.OrderId)

		log.Println(order.OrderId,"-",order.OrderNo,"违约重拍处理结束。。")
	}
}

//处理自收重拍
func SelfReceiveRedo(orderId int,car car,tx *sql.Tx){

	var (
		dealerType int
	)

	//获取拍单信息
	order := getOrder(orderId)

	//拍单是否是自收重拍
	if car.isSelfReceive == 1{
		fmt.Println(order.OrderId , " - " , order.OrderNo , "自收重拍处理。。。")

		stmt,_ := tx.Prepare("UPDATE au_order set confirm_type=? where order_id = ?")
		stmt.Exec(1,order.OrderId)

	}else{
		//自收人拍得非违约重拍的车辆，才打上自收标签
		if car.IsDealerBreach == 0{

			stmt  :=  "SELECT dealer_type FROM au_car_dealer WHERE dealer_id=? limit 1"
			row := db.QueryRow(stmt,order.SuccessDealerId)
			err := row.Scan(&dealerType)
			if err != nil {
				if err == sql.ErrNoRows{
					log.Println("自收车商查询无信息 dealer_id=",order.SuccessDealerId,order)
				}else{
					log.Fatalf("SELECT dealer_type FROM au_car_dealer WHERE dealer_id= %d limit 1 err: %v",order.SuccessDealerId,err)
				}
			}
			if dealerType ==1 {
				fmt.Println(order.OrderId , " - " , order.OrderNo , "自收人拍得车辆处理。。。")

				stmt,_ := tx.Prepare("UPDATE au_cars SET is_self_receive = ?,self_receive_dealer_id = ? where car_id = ?")
				stmt.Exec(car.isSelfReceive,car.SelfReceiveDealerId,car.CarId)
			}
		}
	}
}

