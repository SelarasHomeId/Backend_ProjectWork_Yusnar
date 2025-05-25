package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/config"
	"selarashomeid/internal/factory"
	"selarashomeid/internal/middleware"
	"selarashomeid/pkg/util/general"
	"selarashomeid/pkg/util/trxmanager"

	"github.com/centrifugal/centrifuge"
	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var NodeCentrifugal *centrifuge.Node

func handleLog(e centrifuge.LogEntry) {
	logrus.Infof("%s: %v", e.Message, e.Fields)
}

func InitCentrifugal(ctx context.Context, e *echo.Echo, f *factory.Factory) {
	var err error

	NodeCentrifugal, err = centrifuge.New(centrifuge.Config{
		LogLevel:   centrifuge.LogLevelError,
		LogHandler: handleLog,
	})
	if err != nil {
		panic(err)
	}

	NodeCentrifugal.OnConnecting(func(ctx context.Context, e centrifuge.ConnectEvent) (centrifuge.ConnectReply, error) {
		logrus.Infof("users try connecting: %s", e.ClientID)
		dataContext, err := middleware.JustValidateToken(e.Token)
		if err != nil {
			if err == jwt.ErrTokenExpired {
				return centrifuge.ConnectReply{}, centrifuge.ErrorTokenExpired
			}
			logrus.Infof("error on connecting: %s", err.Error())
			return centrifuge.ConnectReply{}, centrifuge.DisconnectInvalidToken
		}
		return centrifuge.ConnectReply{
			Credentials: &centrifuge.Credentials{
				UserID: strconv.Itoa(dataContext.Auth.ID),
			},
		}, nil
	})

	NodeCentrifugal.OnConnect(func(client *centrifuge.Client) {
		transport := client.Transport()
		logrus.Infof("user %s connected via %s with protocol: %s", client.UserID(), transport.Name(), transport.Protocol())

		client.OnRefresh(func(e centrifuge.RefreshEvent, cb centrifuge.RefreshCallback) {
			logrus.Infof("user %s connection is going to expire, refreshing", client.UserID())
			cb(centrifuge.RefreshReply{ExpireAt: time.Now().Unix() + 60*10}, nil)
		})

		client.OnSubRefresh(func(e centrifuge.SubRefreshEvent, cb centrifuge.SubRefreshCallback) {
			logrus.Infof("user %s connection is going to expire, refreshing sub", client.UserID())
			cb(centrifuge.SubRefreshReply{ExpireAt: time.Now().Unix() + 60*10}, nil)
		})

		// client.OnRPC(func(e centrifuge.RPCEvent, cb centrifuge.RPCCallback) {
		// 	var byteData []byte = nil
		// 	data := make(map[string]interface{})
		// 	err := json.Unmarshal(e.Data, &data)
		// 	if err != nil {
		// 		logrus.Errorln(err)
		// 	}
		// 	functionName := ""
		// 	if data["function"] != nil {
		// 		functionName = data["function"].(string)
		// 	}
		// 	logrus.Infof("user asking for rpc function %s", functionName)
		// 	if functionName == "CountUnreadChat" {
		// 		cpfId := data["cpf_id"].(string)
		// 		data, err := GetUnreadNotification(client.UserID(), cpfId, f.Db)
		// 		if err != nil {
		// 			logrus.Errorf("something wrong: %s", err)
		// 		}

		// 		byteData, err = json.Marshal(data)
		// 		if err != nil {
		// 			logrus.Errorf("something wrong: %s", err)
		// 		}

		// 	}
		// 	cb(centrifuge.RPCReply{Data: byteData}, nil)
		// })

		client.OnSubscribe(func(e centrifuge.SubscribeEvent, cb centrifuge.SubscribeCallback) {
			logrus.Infof("user %s subscribes on %s", client.UserID(), e.Channel)
			if !strings.Contains(e.Channel, "chat-") {
				if e.Channel != client.UserID() {
					cb(centrifuge.SubscribeReply{}, centrifuge.ErrorPermissionDenied)
					logrus.Infof("denied user %s subscribes on %s", client.UserID(), e.Channel)
					return
				}

				cb(centrifuge.SubscribeReply{}, nil)

				userId, _ := strconv.Atoi(client.UserID())
				data, err := GetNotification(userId, f.Db)
				if err != nil {
					logrus.Errorf("something wrong: %s", err)
				}

				byteData, err := json.Marshal(data)
				if err != nil {
					logrus.Errorf("something wrong: %s", err)
				}

				_, err = NodeCentrifugal.Publish(client.UserID(), byteData)
				if err != nil {
					logrus.Errorf("error publishing: %v", err)
				}
			} else if strings.Contains(e.Channel, "chat-") {
				cb(centrifuge.SubscribeReply{}, nil)
			}

		})

		client.OnDisconnect(func(e centrifuge.DisconnectEvent) {
			logrus.Infof("user %s disconnected, disconnect: %s", client.UserID(), e.Disconnect)
		})
	})

	address := fmt.Sprintf("%s:%s", config.Get().Redis.RedisHost, config.Get().Redis.RedisPort)

	redisShardConfigs := []centrifuge.RedisShardConfig{
		{
			Address:  address,
			User:     config.Get().Redis.RedisUser,
			Password: config.Get().Redis.RedisPassword,
		},
		//{Address: "localhost:6380"},
	}
	var redisShards []*centrifuge.RedisShard
	for _, redisConf := range redisShardConfigs {
		redisShard, err := centrifuge.NewRedisShard(NodeCentrifugal, redisConf)
		if err != nil {
			logrus.Fatal(err)
		}
		redisShards = append(redisShards, redisShard)
	}

	broker, err := centrifuge.NewRedisBroker(NodeCentrifugal, centrifuge.RedisBrokerConfig{
		// Use reasonably large expiration interval for stream meta key,
		// much bigger than maximum HistoryLifetime value in Node config.
		// This way stream metadata will expire, in some cases you may want
		// to prevent its expiration setting this to zero value.
		// HistoryMetaTTL: 7 * 24 * time.Hour,

		// And configure a couple of shards to use.
		Shards: redisShards,
	})
	if err != nil {
		logrus.Fatal(err)
	}
	NodeCentrifugal.SetBroker(broker)

	presenceManager, err := centrifuge.NewRedisPresenceManager(NodeCentrifugal, centrifuge.RedisPresenceManagerConfig{
		Shards: redisShards,
	})
	if err != nil {
		logrus.Fatal(err)
	}
	NodeCentrifugal.SetPresenceManager(presenceManager)

	if err := NodeCentrifugal.Run(); err != nil {
		logrus.Fatalf("Error on start centrifuge: %v", err)
	}

	websocketHandler := centrifuge.NewWebsocketHandler(NodeCentrifugal, centrifuge.WebsocketConfig{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	})

	e.GET("/websocket", convert(auth(websocketHandler)))

	go func() {
		for range ctx.Done() {
			ctx2, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			NodeCentrifugal.Shutdown(ctx2)
			logrus.Println("centrifugal is stopped")
			return
		}
	}()
}

func convert(h http.Handler) echo.HandlerFunc {
	return func(c echo.Context) error {
		h.ServeHTTP(c.Response().Writer, c.Request())
		return nil
	}
}

func auth(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		newCtx := centrifuge.SetCredentials(ctx, &centrifuge.Credentials{
			UserID: general.RandSeq(10),
		})

		r = r.WithContext(newCtx)
		h.ServeHTTP(w, r)
	})
}

func GetNotification(usersId int, db *gorm.DB) (map[string]interface{}, error) {

	data := make(map[string]interface{})
	err := db.Table("notifikasi").
		Select("count(*)").
		Where("user_id = ? AND is_read = ?", usersId, false).
		Find(&data).Error
	if err != nil {
		return nil, err
	}
	dataSend := make(map[string]interface{})
	dataSend["count"] = data["count"].(int64)
	if data["count"].(int64) > 0 {
		dataSend["is_new"] = true
	} else {
		dataSend["is_new"] = false
	}

	return dataSend, nil
}

func PublishNotification(usersId int, db *gorm.DB, ctx *abstraction.Context) error {

	channels := NodeCentrifugal.Hub().Channels()
	check := general.StringInSlice(strconv.Itoa(usersId), channels)
	if check {
		if err := trxmanager.New(db).WithTrx(ctx, func(ctx *abstraction.Context) error {

			data := make(map[string]interface{})
			err := db.Table("notifikasi").
				Select("count(*)").
				Where("user_id = ? AND is_read = ?", usersId, false).
				Find(&data).Error
			if err != nil {
				return err
			}
			dataSend := make(map[string]interface{})
			dataSend["count"] = data["count"].(int64)
			dataSend["count"] = data["count"].(int64)
			if data["count"].(int64) > 0 {
				dataSend["is_new"] = true
			} else {
				dataSend["is_new"] = false
			}

			byteData, err := json.Marshal(dataSend)
			if err != nil {
				return err
			}
			_, err = NodeCentrifugal.Publish(strconv.Itoa(usersId), byteData)
			if err != nil {
				logrus.Errorf("error publishing: %v", err)
			}

			return nil
		}); err != nil {
			return err
		}
	}

	return nil
}

func PublishNotificationCustom(channel string, data []byte) error {

	_, err := NodeCentrifugal.Publish(channel, data)
	if err != nil {
		logrus.Errorf("error publishing: %v", err)
	}

	return nil
}

func PublishNotificationWithoutTransaction(usersId int, db *gorm.DB, ctx *abstraction.Context) error {

	channels := NodeCentrifugal.Hub().Channels()
	check := general.StringInSlice(strconv.Itoa(usersId), channels)
	if check {
		data := make(map[string]interface{})
		err := db.Table("notifikasi").
			Select("count(*)").
			Where("user_id = ? AND is_read = ?", usersId, false).
			Find(&data).Error
		if err != nil {
			return err
		}
		dataSend := make(map[string]interface{})
		dataSend["count"] = data["count"].(int64)
		dataSend["count"] = data["count"].(int64)
		if data["count"].(int64) > 0 {
			dataSend["is_new"] = true
		} else {
			dataSend["is_new"] = false
		}

		byteData, err := json.Marshal(dataSend)
		if err != nil {
			return err
		}
		_, err = NodeCentrifugal.Publish(strconv.Itoa(usersId), byteData)
		if err != nil {
			logrus.Errorf("error publishing: %v", err)
		}

		return nil
	}

	return nil
}
