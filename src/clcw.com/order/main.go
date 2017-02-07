package main

import (
	"database/sql"
	"flag"
	//"strconv"
	"fmt"
	"log"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/garyburd/redigo/redis"
	_ "github.com/go-sql-driver/mysql"
	"net/http"
	_ "net/http/pprof"
)

var (
	conf config
	pool *redis.Pool
	db   *sql.DB
)

const (
	MaxOpenDbConnect = 50
	//MaxOpenGoRoutine = 30
)

func main() {

	if _, err := toml.DecodeFile("./conf/config.toml", &conf); err != nil {
		log.Fatal("[CLCW Order Server] toml decode conf err : %s", err.Error())
		return
	}

	//setup log
	f, err := os.OpenFile(conf.Log.File, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("[CLCW Order SERVER] open log failed: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	//setup redis
	redisAddr := fmt.Sprintf("%s:%s", conf.Redis.Host, conf.Redis.Port)
	pool = newRedisPool(redisAddr, conf.Redis.Pswd, conf.Redis.Timeout)
	defer pool.Close()

	//init mysql
	mysql_dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=%s",
		conf.Mysql.User,
		conf.Mysql.Pswd,
		conf.Mysql.Host,
		conf.Mysql.Port,
		conf.Mysql.Database,
		conf.Mysql.Charset,
	)
	log.Printf(mysql_dsn)
	//open doesn't open a connection.
	db, err = sql.Open("mysql", mysql_dsn)
	if err != nil {
		log.Panic("[CLCW Order SERVER] mysql open dsn failed : %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(MaxOpenDbConnect)

	err = db.Ping() //check connection
	if err != nil {
		log.Panic("[CLCW Order SERVER] mysql connect failed : %v", err)
	}

	log.Println("[CLCW Order SERVER] Biz Init Finished.")

	//setup pprof
	go func() {
		http.ListenAndServe("192.168.1.115:6060", nil)
	}()

	//setup command line
	var state string
	var dtime int64
	flag.StringVar(&state, "s", "start", "start:处理开始的订单 end:处理结束的订单")
	flag.Int64Var(&dtime, "d", 0, "0:当前时间戳，大于0: 指定的时间戳")
	flag.Parse()

	if state == "start" {
		if dtime > 0 {
			fmt.Println("运行参数: ", state, dtime)
			orderStartServiceByTime(dtime)
		} else {
			fmt.Println("运行参数: ", state, 0)
			orderStartService()
		}
	} else if state == "end" {
		if dtime > 0 {
			fmt.Println("运行参数: ", state, dtime)
			orderEndServiceByTime(dtime)
		} else {
			fmt.Println("运行参数: ", state, 0)
			orderEndService()
		}
	} else if state == "test" {

		thriftClient()
		//v := RandInt64(1,10000001)
		//fmt.Println(v)
		//car1 := getCar(362)
		//car2 := getCar(371)
		//order1 := getOrder(1455)
		//order2 := getOrder(1456)
		//fmt.Println(car1)
		//fmt.Println(car2)
		//fmt.Println(order1)
		//fmt.Println(order2)

		//card := getDenominationType(80)
		//v := getRandPrice(card)
		//fmt.Println(v)
		////fmt.Println(card)
		//for k, v := range card {
		//	fmt.Println(k,v)
		//}

	}

	select {}
}
