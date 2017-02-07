package main

import (
	"database/sql"
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
		log.Fatal("[CLCW Server] toml decode conf err : %s", err.Error())
		return
	}

	//setup log
	f, err := os.OpenFile(conf.Log.File, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("[CLCW SERVER] open log failed: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	//setup redis
	redisAddr := fmt.Sprintf("%s:%s", conf.Redis.Host, conf.Redis.Port)
	pool = newRedisPool(redisAddr, conf.Redis.Pswd, conf.Redis.Database, conf.Redis.Timeout)
	defer pool.Close()

	//setup mysql
	/*
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
		log.Panic("[CLCW SERVER] mysql open dsn failed : %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(MaxOpenDbConnect)

	err = db.Ping() //check connection
	if err != nil {
		log.Panic("[CLCW SERVER] mysql connect failed : %v", err)
	}
	*/
	log.Println("[CLCW SERVER] Biz Init Finished.")

	//setup pprof
	go func() {
		http.ListenAndServe("127.0.0.1:6061", nil)
	}()

	orderMonitor()
	select {}

	//key := conf.Redis.Dealerlist + fmt.Sprintf("%d", 1777)
	//zrange(key, 0, -1)
}
