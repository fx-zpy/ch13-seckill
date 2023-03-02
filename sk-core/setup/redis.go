package setup

import (
	conf "github.com/fx-zpy/SecKillGolang/pkg/config"
	"github.com/go-redis/redis"

	"log"
)

//初始化redis
func InitRedis() {
	client := redis.NewClient(&redis.Options{
		Addr:     conf.Redis.Host,
		Password: conf.Redis.Password,
		DB:       conf.Redis.Db,
	})

	_, err := client.Ping().Result()
	if err != nil {
		log.Printf("Connect redis failed. Error : %v", err)
	}
	conf.Redis.RedisConn = client
}
