package main

import (
	"crypto/rand"
	"math"
	"math/big"
	"time"
	"sort"
	"fmt"
)

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
 * 返回毫秒
 *
 * return int
 */
func currentTimeMillis() int64 {
	return time.Now().UnixNano() / 1000000
}

/**
 * key 排序
 *
 * return int
 */
func sortKey(list map[int]int64) (keys []int) {
	for k := range list {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return
}

func trace(str string){
	if conf.Log.Debug == 1 {
		fmt.Println(str)
	}
}