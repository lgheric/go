package main

import (
	"encoding/json"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"log"
	"time"

	"database/sql"
	"os/exec"
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
		log.Fatalf("json",res,"解析失败", err)
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
		log.Fatalf("json %s failed: %v \n", err)
	}

	fmt.Printf("scene_id : %d scard : %d \n", order.SceneID, count)
	if count <= 0 {
		oids := getOrderList(order.SceneID)
		if len(oids) > 0 {
			sceneStart(order.SceneID)
			orderWaitBidding(oids, order.SceneID)
			sceneSaddOrder(oids, order.SceneID)
		} else {
			fmt.Printf("%d 会场没有status = 3订单", order.SceneID)
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
		log.Fatalf("json %s failed: %v \n", res, err)
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

				//更新服务费
				commision := getCommision(od.BidBestPrice)
				stmt, _ := tx.Prepare("UPDATE `au_order` SET `success_price` = ?, `success_dealer_id` = ?, `commision` = ? WHERE `order_id` = ?")
				stmt.Exec(od.BidBestPrice, od.BidBestDealerId, commision, od.OrderId)
			} else {
				stmtd, _ := tx.Prepare("INSERT INTO `au_dealer_auctioned` SET `dealer_id` = ?, `order_id` = ?, `createtime` = NOW(), `trade_price` = ?")
				stmtd.Exec(od.BiddingBestDealerId, od.OrderId, od.BiddingBestPrice)

				//保证金
				paimaiRefund(od.OrderId, od.BiddingBestDealerId)

				//更新服务费
				commision := getCommision(od.BiddingBestPrice)
				stmt, _ := tx.Prepare("UPDATE `au_order` SET `success_price` = ?, `success_dealer_id` = ?, `commision` = ? WHERE `order_id` = ?")
				stmt.Exec(od.BiddingBestPrice, od.BiddingBestDealerId, commision, od.OrderId)
			}

			stmtl, _ := tx.Prepare("INSERT INTO `au_order_trace_log_list`(`order_id`, `car_id`, `emp_name`, `action_no`, `action_name`, `createtime`) VALUES (?, ?, ?, ?, ?, ?)")
			stmtl.Exec(od.OrderId, od.CarId, "--", 1008, "竞拍结束", time.Now().UnixNano())

			car := getCar(od.CarId)

			stmtc, _ := tx.Prepare("INSERT INTO `au_car_trace_log_list`(`owner_id`, `car_id`, `emp_name`, `action_no`, `action_name`, `createtime`) VALUES (?, ?, ?, ?, ?, ?)")
			stmtc.Exec(car.OwnerId, car.CarId, "--", 1014, "竞拍结束", time.Now().UnixNano())

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
//处理违约重拍 -- 到平台确认
func breachRedoPlatform(oid int){
	//global $db;
	//$order = $db->where('order_id', $id)->getOne('order');
	//$car = $db->where('car_id', $order['car_id'])->getOne('cars');
	////拍单是否违约重拍
	//if ($car['is_dealer_breach'] == 1) {
	//	echo $order['order_id'] . " - " . $order['order_no'] . "违约重拍处理\n";
	//	$old_order = $db->rawQueryOne("SELECT * FROM `au_order` WHERE `car_id`='{$order['car_id']}' ORDER BY order_id DESC limit 1,1");
	//	$best_price = $order['bidding_best_price'] > $order['bid_best_price'] ? $order['bidding_best_price'] : $order['bid_best_price'];
	//	$success_dealer_id = $order['bidding_best_price'] > $order['bid_best_price'] ? $order['bidding_best_dealer_id'] : $order['bid_best_dealer_id'];
	//	$data = [];
	//	$data['status'] = 5;
	//	$data['success_price'] = $best_price;
	//	$data['success_dealer_id'] = $success_dealer_id;
	//	$data['first_money'] = $old_order['first_money'];
	//	$data['confirm_type'] = 1;
	//	$db->where('order_id', $id)->update('order', $data);
	//}
}

//违约重拍处理
func breachRedo(oid int) {

	var (
		bestPrice       float64
		successDealerId int
		firstPayStatus  int
		tailMoney       float64
	)

	order := getOrder(oid)
	car := getCar(order.CarId)

	// 违约重拍
	if car.IsDealerBreach {
		fmt.Printf("%d - %s 违约重拍处理\n", order.OrderId, order.OrderNo)
		oldOrder := getOrderByCar(order.CarId)

		if order.BiddingBestPrice > order.BidBestPrice {
			bestPrice = order.BiddingBestPrice
			successDealerId = order.BiddingBestDealerId
		} else {
			bestPrice = order.BidBestPrice
			successDealerId = order.BidBestDealerId
		}

		now := time.Now().Format("2006-01-02 15:04:05")

		tx, err := db.Begin()
		if err != nil {
			log.Fatalf("breachRedo mysql transaction begin failed: %v ", err)
		}
		defer tx.Rollback()

		// 判断车辆来源 4S店车源
		if car.CarSource == 1 {

			if car.PayStatus == 2 {
				firstPayStatus = 1
			} else {
				firstPayStatus = 0
			}

			// 先付款后验车
			stmt, _ := tx.Prepare("INSERT INTO `au_proceeds_log` SET `order_id` = ?, `createtime` = '?'")
			stmt.Exec(oid, now)

			stmti, _ := tx.Prepare("UPDATE `au_order` SET `status` = ?, `success_price` = ?, `success_dealer_id` = ?, `return_check_status` = ?, " +
				" `first_money` = ?, `first_pay_status` = ?  WHERE `order_id` = ?")
			stmti.Exec(8, bestPrice, successDealerId, 5, oldOrder.FirstMoney, firstPayStatus, oid)


			// 个人车源
		} else {

			oldPrice := oldOrder.SuccessPrice + oldOrder.CompanySubsidies
			if car.ThreeInOne == 1 {
				if car.PayStatus > 1 {
					firstPayStatus = 1
				} else {
					firstPayStatus = 0
				}

				if bestPrice > oldPrice {
					tailMoney = oldOrder.TailMoney + (bestPrice - oldPrice)
				} else {
					tailMoney = oldOrder.TailMoney
				}

			} else {
				firstPayStatus = 1

				if bestPrice > oldPrice {
					tailMoney = bestPrice
				} else {
					tailMoney = oldPrice
				}
			}

			stmt, _ := tx.Prepare("INSERT INTO `au_proceeds_log` SET `order_id` = ?, `createtime` = '?'")
			stmt.Exec(oid, now)

			stmti, _ := tx.Prepare("UPDATE `au_order` SET `status` = ?, `success_price` = ?, `success_dealer_id` = ?, `tail_money` = ?, " +
				" `first_money` = ?, `first_pay_status` = ?  WHERE `order_id` = ?")
			stmti.Exec(8, bestPrice, successDealerId, tailMoney, oldOrder.FirstMoney, firstPayStatus, oid)

		}

		errc := tx.Commit()
		if errc != nil {
			log.Fatalf("breachRedo mysql transaction commit failed: %v ", errc)
		}

		// 违约重拍这里发券
		var (
			branchId   int
			activityId int
		)

		branchId = getBranchId(car.LocationArea)

		// 发放抽奖卡券
		activityId = isHaveActivity(now, branchId, 1)
		if activityId > 0 {
			sendCard(oid, activityId)
		}

		// 发放抽代金券卡券
		activityId = isHaveActivity(now, branchId, 2)
		if activityId > 0 {
			sendCard(oid, activityId)
		}

	}

}

//处理自收重拍
func SelfReceiveRedo(id int){
	//global $db;
	//$order = $db->where('order_id', $id)->getOne('order');
	//$car = $db->where('car_id', $order['car_id'])->getOne('cars');
	////拍单是否自收重拍
	//$success_dealer_id = $order['bidding_best_price'] > $order['bid_best_price'] ? $order['bidding_best_dealer_id'] : $order['bid_best_dealer_id'];
	//if ($car['is_self_receive'] == 1) {
	//echo $order['order_id'] . " - " . $order['order_no'] . "自收重拍处理\n";
	//$best_price = $order['bidding_best_price'] > $order['bid_best_price'] ? $order['bidding_best_price'] : $order['bid_best_price'];
	////判断车辆来源
	//$data = [];
	//$data['status'] = 5;
	//$data['success_price'] = $best_price;
	//$data['success_dealer_id'] = $success_dealer_id;
	//$data['confirm_type'] = 1;
	//$db->where('order_id', $id)->update('order', $data);
	//} else {
	////自收人拍得非违约重拍的车辆，才打上自收标签
	//if ($car['is_dealer_breach'] == 0) {
	//echo $order['order_id'] . " - " . $order['order_no'] . "自收人拍得车辆处理\n";
	//$dealer = $db->where('dealer_id', $success_dealer_id)->getOne('car_dealer', 'dealer_type');
	//if ($dealer['dealer_type'] == 1) {
	//$db->where('car_id', $order['car_id'])->update('cars', [
	//'is_self_receive' => 1,
	//'self_receive_dealer_id' => $success_dealer_id
	//]);
	//}
	//}
	//}
}

func update_trace_log(orderId int){

	stmt := "SELECT `car_id` FROM `au_order` WHERE `order_id` = ? LIMIT 1"
	rows := db.QueryRow(stmt,orderId)
	var car_id int
	err := rows.Scan(&car_id)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Println("找不到车源 car_id:",car_id)
		} else {
			log.Fatalf("SELECT `car_id` FROM `au_order` WHERE `order_id` = %d exec failed: %v ", orderId,err)
		}
	}

	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("update_trace_log mysql transaction begin failed: %v ", err)
	}
	defer tx.Rollback()

	stmtl, _ := tx.Prepare("INSERT INTO `au_order_trace_log_list`(`order_id`, `car_id`, `emp_name`, `action_no`, `action_name`, `createtime`) VALUES (?, ?, ?, ?, ?, ?)")
	stmtl.Exec(orderId, order.CarId, "--", 1007, "开始竞拍", time.Now().UnixNano())


	//$now = microtime_time();
	//$orderInfo = $db->where('order_id', $id)->getOne('order','car_id');
	//$db->insert('order_trace_log_list',[
	//'order_id' => $id,
	//'car_id'   => $orderInfo['car_id'],
	//'emp_name' => '--',
	//'action_no' => 1008,
	//'action_name' => '竞拍结束',
	//'createtime' => $now
	//]);
	//$carInfo = $db->where('car_id', $orderInfo['car_id'])->getOne('cars','owner_id,sid');
	//$db->insert('car_trace_log_list',[
	//'owner_id'   => $carInfo['owner_id'],
	//'car_id' => $orderInfo['car_id'],
	//'emp_name' => '--',
	//'action_no' => 1014,
	//'action_name' => '竞拍结束',
	//'createtime' =>$now
	//]);

}

//更新车源状态
func update_car_source(sid int){

	stmt := "SELECT `status` FROM `au_car_source` WHERE `sid` = ? LIMIT 1"
	rows := db.QueryRow(stmt, sid)

	var status int
	err := rows.Scan(&status)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Println("找不到车源sid:",sid)
		} else {
			log.Fatalf("select status from  au_car_source where sid= %d 543 mysql fetch result failed: %v ", sid,err)
		}
	}

	if status == 200 {
		stmt = "UPDATE `au_car_source` SET `status` = 300 WHERE `sid` = ?"
		_, err := db.Exec(stmt, sid)
		if err != nil {
			log.Printf("UPDATE `au_car_source` SET `status` = 300 WHERE `sid` = %d exec failed: %v \n", sid,err)
		}
		fmt.Printf("更新车源 %d 状态为300 待确认\n", sid)
	}
}
func del_autobidding_redis_key(orderId int){

	conn := pool.Get()
	defer conn.Close()

	if _, err := conn.Do("SELECT", 4); err != nil {
		log.Fatalf("Redis 选择库失败。Err:  %s ", err)
		return
	}
	//autobidding_order_list
	queueName := conf.Redis.AutoOrderLs + string(orderId)
	_, err := redis.String(conn.Do("srem", queueName))
	if err != nil {
		fmt.Printf("%s queue srem failed: %s ", queueName, err)
	}

	//删除所有设置智能出价的车商
	//autobidding_dealer_list_拍单ID
	queueName = conf.Redis.AutoDealerLs + string(orderId)
	_, err = redis.String(conn.Do("del", queueName))
	if err != nil {
		fmt.Printf("%s queue del failed: %s ", queueName, err)
	}

	//autobidding_dealer_info_拍单ID_*
	queueName = conf.Redis.AutoDealerIf + string(orderId)+"_*"
	cmd := exec.Command("`which sh`", "-c", "`redis-cli keys"+ queueName +"|xargs redis-cli del`")
	_, err = cmd.StdoutPipe()
	if err != nil {
		fmt.Println(queueName,"删除失败。","错误: " + err.Error())
	}

	//autobidding_order_info_拍单ID
	queueName = conf.Redis.AutoOrderIf + string(orderId)
	_, err = redis.String(conn.Do("del", queueName))
	if err != nil {
		fmt.Printf("%s queue del failed: %s ", queueName, err)
	}
	//autobidding_order_bidding_拍单ID
	queueName = conf.Redis.AutoOrderBi + string(orderId)
	_, err = redis.String(conn.Do("del", queueName))
	if err != nil {
		fmt.Printf("%s queue del failed: %s ", queueName, err)
	}
}