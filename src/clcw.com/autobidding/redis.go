package main

import (
	"github.com/garyburd/redigo/redis"
	"log"
	"strconv"
	"time"
)

// redis
func newRedisPool(server string, password string, database string, timeout int) *redis.Pool {
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

			if  database != "" {
				if _, err := c.Do("SELECT", database); err != nil {
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
		log.Fatalf("%s exists failed: %s ", key, err)
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
 * redis smembers
 *
 */
func smembers(key string) (ids []int) {
	conn := pool.Get()
	defer conn.Close()

	ids, _ = redis.Ints(conn.Do("SMEMBERS", key))
	return
}

/**
 * redis get
 *
 */
func redisGet(key string) (str string) {
	conn := pool.Get()
	defer conn.Close()

	str, _ = redis.String(conn.Do("GET", key))
	return
}

/**
 * redis zrange
 *
 */
func zrange(key string, start int64, stop int64) (list []int) {
	conn := pool.Get()
	defer conn.Close()

	list, _ = redis.Ints(conn.Do("ZRANGE", key, start, stop))
	return
}

/**
 * redis zrange
 *
 */
/*
func zrange(key string, start int64, stop int64) (list map[int]int64) {
	conn := pool.Get()
	defer conn.Close()

	arr, _ := redis.IntMap(conn.Do("ZRANGE", key, start, stop, "WITHSCORES"))

	list = make(map[int]int64)
	for k, v := range arr {
		t, _ := strconv.Atoi(k)
		list[v] = int64(t)

	}
	return
}
 */

func getDealerId(key string) (dealerId int) {
	conn := pool.Get()
	defer conn.Close()

	arr, err := redis.Strings(conn.Do("ZRANGE", key, 0, 0))

	if err != nil {
		log.Printf("%s redis sorted set is empty failed: %v \n", key, err)
	}
	if len(arr) > 0 {
		dealerId, _ = strconv.Atoi(arr[0])
	}
	return
}
