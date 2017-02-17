package main

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"log"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"
)

// redis
func newRedisPool(server string, password string, timeout int) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     5,
		IdleTimeout: time.Duration(timeout) * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", server)
			if err != nil {
				return nil, err
			}

			if password != "" {
				if _, err := c.Do("AUTH", password); err != nil {
					c.Close()
					return nil, err
				}
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}

/**
 * 计算交易服务费
 *
 * @param float64 price 初始价格
 * @return float64 p 交易服务费
 */
func getCommision(price float64) (p float64) {
	p = price * conf.Fee.Percentage
	if p <= float64(conf.Fee.Minfee) {
		p = float64(conf.Fee.Minfee)
	} else if p >= float64(conf.Fee.Maxfee) {
		p = float64(conf.Fee.Maxfee)
	} else {
		p = round(p, 0)
	}
	return
}

/**
 * 四舍五入
 *
 */
func round(f float64, n int) float64 {
	pow := math.Pow10(n)
	return math.Trunc((f+0.5/pow)*pow) / pow
}

/**
 * 生成随机数
 *
 */
func RandInt64(min, max int64) int64 {
	maxBigInt := big.NewInt(max)
	i, _ := rand.Int(rand.Reader, maxBigInt)
	if i.Int64() < min {
		RandInt64(min, max)
	}
	return i.Int64()
}

/**
 * redis rpop
 *
 */
func rpop(queue string) (s string, e error) {
	conn := pool.Get()
	defer conn.Close()

	arr, err := redis.Strings(conn.Do("BRPOP", queue, 0))
	if len(arr) < 2 {
		e = err
	} else {
		s = arr[1]
	}
	return
}

/**
 * redis lpush
 *
 */
func lpush(queue string, m []byte) error {
	conn := pool.Get()
	defer conn.Close()

	_, e := conn.Do("LPUSH", queue, m)
	return e
}

/**
 * redis exists
 *
 */
func exists(key string) (b bool, err error) {
	conn := pool.Get()
	defer conn.Close()

	b, err = redis.Bool(conn.Do("EXISTS", key))
	if err != nil {
		log.Fatalf("%s queue exists failed: %s ", key, err)
	}
	return
}

/**
 * redis lock
 *
 */
func redisLock(key string) (bool bool) {
	conn := pool.Get()
	defer conn.Close()

	bool, err := redis.Bool(conn.Do("SETNX", key, 1))
	if err != nil {
		log.Fatalf("%s redis key lock failed: %v ", key, err)
	}

	if !bool {
		time.Sleep(time.Second)
		//redisLock(key)
		bool, _ = redis.Bool(conn.Do("SETNX", key, 1))
	}
	return
}

/**
 * redis unlock
 *
 */
func redisUnLock(key string) error {
	conn := pool.Get()
	defer conn.Close()

	_, e := conn.Do("DEL", key)
	return e
}

/**
 * 字符串时间格式化成时间戳
 *
 * return int
 */
func formatTime(dateStr string) (s int64, err error) {
	timestamp, err := time.ParseInLocation("2006-01-02 15:04:05", dateStr, time.Local)
	s = timestamp.Unix()
	return
}

/**
 * 场次开始
 *
 * @param int sid 场次ID
 */
func sceneStart(sid int) {
	var status int
	stmt := "SELECT status FROM au_order_scene WHERE scene_id = ?"
	row := db.QueryRow(stmt,sid)
	err := row.Scan(&status)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Fatalf("没找到场 %d",sid)
		}else{
			log.Fatalf("获取场信息失败:stmt= %s %d",stmt,sid)
		}
	}
	if status ==1 {
		stmt := "UPDATE `au_order_scene` SET `status` = 2 WHERE `scene_id` = ?"
		_, err := db.Exec(stmt, sid)
		if err != nil {
			log.Printf("Mysql exec failed: %v \n", err)
			log.Printf("Stmt: %s \n", stmt)
		}
		fmt.Printf("场次 %d 开始 \n", sid)

	}
}

/**
 * 场次结束
 *
 * @param int sid 场次ID
 */
func sceneEnd(sid int) {
	//判断场内是否还有拍单
	var count int
	stmt := "SELECT COUNT(*) FROM au_order WHERE scene_id = ? AND status in(3,301,4)"
	row := db.QueryRow(stmt,sid)
	err := row.Scan(&count)
	if err == nil {
		if count <=0 {
			_, err := db.Exec("UPDATE `au_order_scene` SET `status` = 3, `endtime` = now() WHERE `scene_id` = ?", sid)
			if err != nil {
				log.Printf("mysql exec failed: %v \n", err)
			}
			fmt.Printf("场次 %d 结束 \n", sid)
		}
	}else{
		log.Fatalf("致命错误：%v stmt:%s",err,stmt)
	}
}

/**
 * 场次redis set拍单数量
 *
 * @param odis array
 * @param sid int
 */
func sceneSaddOrder(oids []int, sid int) {
	conn := pool.Get()
	defer conn.Close()

	for _, oid := range oids {

		bool, err := redis.Bool(conn.Do("SADD", conf.Redis.Scene+fmt.Sprintf("%d", sid), oid))
		if err != nil || !bool {
			log.Printf("Redis SADD %s %d 命令执行失败。 err:%v \n", conf.Redis.Scene+fmt.Sprintf("%d", sid), oid,err)
		}else{
			log.Printf("Redis SADD %s %d 命令执行成功！ \n", conf.Redis.Scene+fmt.Sprintf("%d", sid), oid)
		}
	}
}

/**
 * 移出一个场次redis set拍单
 *
 * @param oid int
 * @param sid int
 */
func sceneSremOrder(oid int, sid int) error {
	conn := pool.Get()
	defer conn.Close()

	_, err := conn.Do("SREM", conf.Redis.Scene+fmt.Sprintf("%d", sid), oid)
	return err
}

/**
 * 场次拍单都变更为待竞拍
 *
 * @param odis array
 * @param sid int
 */
func orderWaitBidding(oids []int, sid int) {
	ids := []string{}
	for _, v := range oids {
		ids = append(ids, strconv.Itoa(v))
	}
	//stmt := fmt.Sprintf("UPDATE `au_order` SET `status` = 301 WHERE `order_id` in (%s)", strings.Join(ids, ","))
	stmt := fmt.Sprintf("UPDATE `au_order` SET `status` = 301 WHERE status=3 AND `scene_id` = %d", sid)
	fmt.Printf("DEBUG:%s \n",stmt)
	_, err := db.Exec(stmt)
	if err != nil {
		log.Printf("Mysql exec failed on util line 139 : %v \n", err)
	}

	fmt.Printf("|%d|:场内全部拍单列表:%s \n", sid, strings.Join(ids, ","))
}

/**
 * 一个场次下面的所有订单ID
 *
 * @param sid int
 * return array
 */
func getOrderList(sid int) (oids []int) {
	oid := 0
	rows, err := db.Query("SELECT order_id FROM au_order WHERE scene_id = ? AND status = 3 ORDER BY rank ASC", sid)
	if err != nil {
		log.Printf("In util line 149 mysql query failed: %v \n", err)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&oid)
		if err != nil {
			log.Fatalf("In util line 156 mysql fetch result failed: %v ", err)
		}
		oids = append(oids, oid)
	}
	return
}

/**
 * 获取一个订单数据
 *
 * @param oid int
 * return order
 */
func getOrder(oid int) (od order) {

	stmt := "SELECT order_id, order_no, scene_id, car_id, status, rank, is_timing_order, " +
		"bid_start_time, bidding_start_time, bidding_end_time, est_elapsed_time, " +
		"act_elapsed_time, bidding_best_dealer_id, bid_best_dealer_id, bidding_best_price, " +
		"bid_best_price, first_money, success_price, company_subsidies, tail_money ,success_dealer_id " +
		"FROM au_order WHERE order_id = ? "
	rows := db.QueryRow(stmt, oid)
	err := rows.Scan(&od.OrderId, &od.OrderNo, &od.SceneId, &od.CarId, &od.Status, &od.Rank,
		&od.IsTimingOrder, &od.BidStartTime, &od.BiddingStartTime, &od.BiddingEndTime,
		&od.EstElapsedTime, &od.ActElapsedTime, &od.BiddingBestDealerId, &od.BidBestDealerId, &od.BiddingBestPrice,
		&od.BidBestPrice, &od.FirstMoney, &od.SuccessPrice, &od.CompanySubsidies, &od.TailMoney,&od.SuccessDealerId)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("%d can't find order", oid)
		} else {
			log.Printf("stmt:%s", stmt)
			log.Fatalf("getOrder 298 mysql fetch result failed: %v ", err)
		}
	}
	return
}

/**
 * 获取一个车辆数据
 *
 * @param cid int
 * return car
 */
func getCar(cid int) (car car) {

	stmt := "SELECT car_id,car_no,sid,sno,owner_id,is_dealer_breach,car_source,pay_status,delivery_mode,three_in_one,location_area,is_self_receive FROM au_cars WHERE car_id = ? "
	rows := db.QueryRow(stmt, cid)
	err := rows.Scan(&car.CarId, &car.CarNo,&car.Sid,&car.Sno,&car.OwnerId, &car.IsDealerBreach, &car.CarSource, &car.PayStatus, &car.DeliveryMode, &car.ThreeInOne, &car.LocationArea,&car.IsSelfReceive)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("%d can't find car", cid)
		} else {
			log.Fatalf("getCar 320 mysql fetch result failed: %v ", err)
		}
	}
	return
}

/**
 * 根据车ID 获取一个订单数据
 *
 * @param cid int
 * return order
 */
func getOrderByCar(carId int) (od order) {

	stmt := "SELECT order_id, order_no, scene_id, car_id, status, rank, is_timing_order, " +
		"bid_start_time, bidding_start_time, bidding_end_time, est_elapsed_time, " +
		"act_elapsed_time, bidding_best_dealer_id, bid_best_dealer_id, bidding_best_price, " +
		"bid_best_price, first_money, success_price, company_subsidies, tail_money " +
		"FROM au_order WHERE car_id = ? ORDER BY order_id DESC"
	rows := db.QueryRow(stmt, carId)
	err := rows.Scan(&od.OrderId, &od.OrderNo, &od.SceneId, &od.CarId, &od.Status, &od.Rank,
		&od.IsTimingOrder, &od.BidStartTime, &od.BiddingStartTime, &od.BiddingEndTime,
		&od.EstElapsedTime, &od.ActElapsedTime, &od.BiddingBestDealerId, &od.BidBestDealerId, &od.BiddingBestPrice,
		&od.BidBestPrice, &od.FirstMoney, &od.SuccessPrice, &od.CompanySubsidies, &od.TailMoney)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("%d can't find order", carId)
		} else {
			log.Fatalf("getOrderByCar 349 mysql fetch result failed: %v ", err)
		}
	}
	return
}

/**
 * 解除保证金
 *
 * @param oid int 订单ID
 * @param did int 车商ID
 */
func paimaiRefund(oid int, did int) {

	var (
		blId       int
		dealerId   int
		occurMoney float64
		title      string
	)

	rows, err := db.Query("SELECT `bl_id`,`dealer_id`,`occur_money` FROM `au_dealer_bail_log` WHERE `order_id` = ? AND `dealer_id` != ? AND `use_type` = 2 AND is_free = 0 ", oid, did)
	if err != nil {
		log.Printf("mysql query failed: %v \n", err)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&blId, &dealerId, &occurMoney)
		if err != nil {
			log.Fatalf("mysql fetch result failed: %v ", err)
		}

		dbl := new(dealerBailLog)
		dbl.blId = blId
		dbl.dealerId = dealerId
		dbl.occurMoney = occurMoney

		stmt := fmt.Sprintf("SELECT CONCAT('解冻出价保证金，拍品【',car.car_no,',',%d,'】 ',' 。') AS title FROM  au_order o INNER JOIN au_cars car ON car.car_id = o.`car_id` WHERE  o.`order_id` = %d", oid, oid)
		rows := db.QueryRow(stmt)
		err = rows.Scan(&title)
		if err != nil {
			log.Fatalf("mysql fetch result failed: %v ", err)
		}

		updateGuarantee(oid, title, dbl)

	}
}

/**
 * 更新保证金
 *
 */
func updateGuarantee(oid int, title string, dbl *dealerBailLog) (err error) {

	var bailAmount float64
	key := conf.Redis.Dealerlock + fmt.Sprintf("%d", dbl.dealerId)

	keyBool, err := exists(key)
	if !keyBool {
		lockBool := redisLock(key)
		if lockBool {
			tx, err := db.Begin()
			if err != nil {
				log.Fatalf("mysql transaction begin failed: %v ", err)
			}
			defer tx.Rollback()

			_, err = tx.Exec("UPDATE `au_car_dealer` SET shortname=shortname WHERE `dealer_id` = ? LIMIT 1", dbl.dealerId)

			rows := tx.QueryRow("SELECT `bail_amount` FROM `au_car_dealer` WHERE `dealer_id` = ?", dbl.dealerId)
			err = rows.Scan(&bailAmount)
			if err != nil {
				log.Fatalf("mysql fetch result failed: %v ", err)
			}

			_, err = tx.Exec("UPDATE `au_car_dealer` SET shortname=shortname WHERE `dealer_id` = ? LIMIT 1", dbl.dealerId)

			stmt, _ := tx.Prepare("INSERT INTO `au_dealer_bail_log` SET `dealer_id` = ?, `order_id` = ?, `use_time` = NOW(), `use_type` = 3, `use_detail` = ?, `occur_money` = ?, `remain_amount`= ?, `free_bl_id`= ?, `createtime` = NOW()")
			stmt.Exec(dbl.dealerId, oid, title, dbl.occurMoney, bailAmount+dbl.occurMoney, dbl.blId)

			_, err = tx.Exec("UPDATE `au_dealer_bail_log` SET `is_free` = 1 WHERE bl_id = ?", dbl.blId)

			_, err = tx.Exec("UPDATE `au_car_dealer` SET `freeze_amount` = `freeze_amount` - ? , `bail_amount` = `bail_amount` + ? WHERE `dealer_id` = ?", dbl.occurMoney, dbl.occurMoney, dbl.dealerId)

			tx.Commit()
			redisUnLock(key)
			fmt.Printf("车商：%d 冻结保证金解除 \n", dbl.dealerId)
		}

	} else {
		time.Sleep(time.Second)
		updateGuarantee(oid, title, dbl)
	}
	return
}

/**
 * 获取所在分公司
 *
 */
func getBranchId(cityCode int) (branchId int) {
	stmt := "SELECT `branch_id` FROM `au_branch_city` WHERE `city_code` = ? "
	rows := db.QueryRow(stmt, cityCode)
	err := rows.Scan(&branchId)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Println("无所在分公司")
			branchId = 0
		} else {
			log.Fatalf("getBranchId 447 mysql fetch result failed: %v ", err)
		}
	}
	return
}

/**
 * 当前时间有没有对应活动类型的抽奖活动
 *
 * @return  int 活动id
 */
func isHaveActivity(now string, branchId int, activityType int) (aid int) {
	stmt := fmt.Sprintf("SELECT `aid` FROM `au_activity` WHERE `starttime` <= '%s' AND `endtime` >= '%s' AND `branch_id` = %d AND `type` = %d ORDER BY createtime DESC ", now, now, branchId, activityType)
	rows := db.QueryRow(stmt)
	err := rows.Scan(&aid)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Println("无活动")
			aid = 0
		} else {
			log.Fatalf("isHaveActivity 469 mysql fetch result failed: %v ", err)
		}
	}
	return
}

/**
 * 获取指定活动下所有类型卡卷的数量
 *
 * @param int activityId
 * @return  count int, price float64
 */
func getDenominationType(activityId int) (card map[int]float64) {

	var (
		num   int
		price float64
	)

	card = make(map[int]float64)

	rows, err := db.Query("SELECT count(0) AS `num`, `price` FROM `au_scratch_card` WHERE `a_id` = ? GROUP BY price ORDER BY price DESC", activityId)
	if err != nil {
		log.Printf("getDenominationType 484 mysql query failed: %v \n", err)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&num, &price)
		if err != nil {
			log.Fatalf("getDenominationType 491 mysql fetch result failed: %v ", err)
		}
		card[num] = price
	}
	return
}

/**
 * 随机算法
 *
 * @return  String 随机出来的面额
 **/
func getRandPrice(card map[int]float64) (price float64) {

	if len(card) <= 0 {
		return
	}

	var (
		allNum int
		i      int
		total  float64
	)

	type tmp_a struct {
		num   float64
		price float64
	}

	type tmp_b struct {
		start_num float64
		end_num   float64
		price     float64
	}

	for key, _ := range card {
		allNum = allNum + key
	}
	ta := make(map[int]tmp_a)
	for key, val := range card {
		ca := new(tmp_a)
		ca.num = math.Ceil((float64(key) / float64(allNum)) * 1000000)
		ca.price = val
		ta[i] = *ca
		total += ca.num
		i++
	}
	tb := make(map[int]tmp_b)
	for key, val := range ta {
		cb := new(tmp_b)
		cb.start_num = tb[key-1].end_num
		cb.end_num = val.num + tb[key-1].end_num
		cb.price = val.price
		tb[key] = *cb
	}
	randNum := RandInt64(1, int64(total))
	for _, val := range tb {
		if float64(randNum) > val.start_num && float64(randNum) < val.end_num {
			price = val.price
		}
	}
	return

}

/**
 * 获取一个活动ID
 *
 */
func getCard(price float64, aid int) (cid int) {
	if price <= 0 || aid <= 0 {
		return
	}
	stmt := "SELECT `cid` FROM `au_scratch_card` WHERE `a_id` = ? AND price = ? AND order_id = 0 LIMIT 1"
	rows := db.QueryRow(stmt, aid, price)
	err := rows.Scan(&cid)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Println("can't find coupon")
			aid = 0
		} else {
			log.Fatalf("getCard 570 mysql fetch result failed: %v ", err)
		}
	}
	return
}

/**
 * 发卡
 *
 * @param oid int
 * @param aid int
 */
func sendCard(oid int, aid int) {
	if oid <= 0 {
		return
	}

	card := getDenominationType(aid)
	if len(card) <= 0 {
		fmt.Printf("活动 ： %d 没有符合条件的卡 \n", aid)
		return
	}

	key := conf.Redis.Cardlock
	keyBool, _ := exists(key)
	if !keyBool {
		againSearch(card, aid, oid)
	} else {
		time.Sleep(time.Second)
		againSearch(card, aid, oid)
	}

}

/**
 * 该面值卡券已全部分发出去，需要重新去池子里面找
 *
 * @return
 */
func againSearch(card map[int]float64, aid int, oid int) {
	price := getRandPrice(card)
	if price > 0 {
		cid := getCard(price, aid)
		fmt.Println(cid)
		if cid > 0 {
			key := conf.Redis.Cardlock
			redisLock(key)
			bindingCard(cid, oid)
			redisUnLock(key)
		} else {
			newCard := make(map[int]float64)
			for key, val := range card {
				if val != price {
					newCard[key] = val
				}
			}
			if len(newCard) <= 0 {
				fmt.Printf("活动 ： %d 没有符合条件的卡 \n", aid)
				return
			}
			againSearch(newCard, aid, oid)
		}
	}
}

/**
 *  给卡券绑定订单
 *
 */
func bindingCard(cid int, oid int) {
	if cid <= 0 || oid <= 0 {
		return
	}
	_, err := db.Exec("UPDATE `au_scratch_card` SET `order_id` = ? WHERE `cid` = ?", oid, cid)
	if err != nil {
		log.Printf("bindingCard 643 mysql exec failed: %v \n", err)
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
		row.Scan(&firtMoney)

		stmt1, _ := tx.Prepare("UPDATE au_order SET first_money=?,confirm_type = ? WHERE order_id = ?")
		stmt1.Exec(1,firtMoney,order.OrderId)

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
	if car.IsSelfReceive == 1{
		fmt.Println(order.OrderId , " - " , order.OrderNo , "自收重拍处理。。。")

		stmt,_ := tx.Prepare("UPDATE au_order set confirm_type=? where order_id = ?")
		stmt.Exec(1,order.OrderId)

	}else{
		//自收人拍得非违约重拍的车辆，才打上自收标签
		if car.IsDealerBreach == 0 {

			stmt  :=  "SELECT dealer_type FROM au_car_dealer WHERE dealer_id=? limit 1"
			row := db.QueryRow(stmt,order.SuccessDealerId)
			row.Scan(&dealerType)

			if dealerType ==1 {
				fmt.Println(order.OrderId , " - " , order.OrderNo , "自收人拍得车辆处理。。。")

				stmt,_ := tx.Prepare("UPDATE au_cars SET is_self_receive = ?,self_receive_dealer_id = ? where car_id = ?")
				stmt.Exec(1,order.SuccessDealerId,car.CarId)
			}
		}
	}
}
