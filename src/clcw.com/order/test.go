package main


import(
	"fmt"
	"time"
	"github.com/garyburd/redigo/redis"
)

type DealerInfo struct{
	dealerId int
	maxPrice float64
	bidCount int
	createTime int
	mobile string
	jpushId string
	isPushed int
}

type Error interface {
	Error() string
}

func main() {
	time.Now().Unix()

	conn := pool.Get()
	defer conn.Close()

	conn.Do("select",4)
	queueName := "autobidding_dealer_info_2687_*"
	var err Error
	mapReply, err := redis.StringMap(conn.Do("keys", queueName),err)
	for k,v := range mapReply{
		fmt.Println("k:",k,"v:",v,"Err:",err)

	}



}
