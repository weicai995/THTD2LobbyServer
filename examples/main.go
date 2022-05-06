package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/lestrrat/go-file-rotatelogs"
	"github.com/pkg/errors"
	"github.com/rifflock/lfshook"
	log "github.com/sirupsen/logrus"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
	"wsserver/configs"
	"wsserver/iserverface"
	"wsserver/server"
	"wsserver/zlog"
)

var (
	configFile string
)

func ConfigLocalFilesystemLogger(logPath string, logFileName string, maxAge time.Duration, rotationTime time.Duration) {
	baseLogPaht := path.Join(logPath, logFileName)
	writer, err := rotatelogs.New(
		baseLogPaht+".%Y%m%d%H%M",
		rotatelogs.WithLinkName(baseLogPaht),      // 生成软链，指向最新日志文件
		rotatelogs.WithMaxAge(maxAge),             // 文件最大保存时间
		rotatelogs.WithRotationTime(rotationTime), // 日志切割时间间隔
	)

	src, err := os.OpenFile(os.DevNull, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		fmt.Println("err", err)
	}
	writer1 := bufio.NewWriter(src)
	log.SetOutput(writer1)

	if err != nil {
		log.Errorf("config local file system logger error. %+v", errors.WithStack(err))
	}
	lfHook := lfshook.NewHook(lfshook.WriterMap{
		log.DebugLevel: writer,
		log.InfoLevel:  writer,
		log.WarnLevel:  writer,
		log.ErrorLevel: writer,
		log.FatalLevel: writer,
		log.PanicLevel: writer,
	}, &log.JSONFormatter{})
	log.AddHook(lfHook)
}

func initCmd() {
	flag.StringVar(&configFile, "config", "./config.json", "where load config json")
	flag.Parse()
}

// 	WebSocket服务端
//ping test 自定义路由
type PingRouter struct {
	server.BaseRouter
}

//Ping Handle 这是一个 例子
func (this *PingRouter) Handle(request iserverface.IRequest) {

	err := request.GetConnection().SendMessage(request.GetMsgType(), []byte("ping...ping...ping"))
	if err != nil {
		zlog.Error(err)
	}
}

// 接收客户端消息 把消息拼入缓存中
type TD2拼接Router struct {
	server.BaseRouter
}

func (this *TD2拼接Router) Handle(request iserverface.IRequest) {

	server.RoomState_locked.RLock()
	rs := server.RoomState_locked.M[0]
	server.RoomState_locked.RUnlock()
	if !rs {
		return
	}

	//拼接一下消息
	//NextMessage = append(NextMessage, idMap[conn])
	server.NextMessage = append(server.NextMessage, byte(0))
	server.NextMessage = append(server.NextMessage, server.Int32ToBytes(len(request.GetData()))...)
	server.NextMessage = append(server.NextMessage, request.GetData()...)
}

//
type QueryRouter struct {
	server.BaseRouter
}

type Response struct {
	ResponseType string
	Data         string
}

func (this *QueryRouter) Handle(request iserverface.IRequest) {
	var m server.JsonRequest
	err2 := json.Unmarshal(request.GetData(), &m) //转到通用接口中
	if err2 != nil {
		zlog.Error(err2)
	}

	/*	fmt.Println(
		server.GWServer.GetConnMgr().Len())
	*/
	//var ids []uint64 = server.GWServer.GetConnMgr().GetAllHostIds()

	var ids []uint64 = server.GWServer.GetConnMgr().QueryRooms()

	//var idsExcept []uint64
	var builder strings.Builder
	//需要删除请求客户端自己的id

	//判断是否需要加逗号
	var needComma int = 0
	for _, id := range ids {
		if id != request.GetConnection().GetConnID() {
			//idsExcept = append(idsExcept, id)
			if needComma == 1 {
				builder.WriteString(",")
			}

			builder.WriteString(strconv.FormatUint(id, 10))
			needComma = 1

		}
	}
	response := Response{"hostlist", builder.String()}
	jsonbyte, err2 := json.Marshal(response)
	if err2 != nil {
		zlog.Error(err2)
	}

	err := request.GetConnection().SendMessage(request.GetMsgType(), jsonbyte)
	if err != nil {
		zlog.Error(err)
	}
}

//处理加入房间请求
type JoinRouter struct {
	server.BaseRouter
}

func (this *JoinRouter) Handle(request iserverface.IRequest) {
	var m server.JsonRequest
	err2 := json.Unmarshal(request.GetData(), &m) //转到通用接口中
	if err2 != nil {
		zlog.Error(err2)
	}

	origianlhostId := m.Params[0].(float64)
	hostId := uint64(origianlhostId)
	/*hostId, err3 := strconv.ParseUint(""+origianlhostId, 10, 64)
	if err3 != nil {
		zlog.Error(err3)
	}*/
	//需要去找房间
	var conn iserverface.IConnection
	var err error
	if ok, _ := server.GWServer.GetConnMgr().QueryRoomStatus(hostId); ok {
		//如果显示房间可用 则直接加入房间 新玩家id为 connection id
		fmt.Println(hostId)
		conn, err = server.GWServer.GetConnMgr().Get(hostId)
		if err != nil {
			zlog.Error(err)
		}

		var selfId = request.GetConnection().GetConnID()
		//去修改一下id
		m.Params[1].(map[string]interface{})["id"] = strconv.FormatUint(selfId, 10) //strconv.Itoa(len)

		b, _ := json.Marshal(m.Params[1])
		//m.Params[1] 是playerinfo
		response := Response{"create", string(b)}
		jsonbyte, err2 := json.Marshal(response)
		if err2 != nil {
			zlog.Error(err2)
		}

		server.GWServer.GetConnMgr().PushAllInRoom(hostId, jsonbyte)

		/*//通知一下 加入者 修改id
		response2 := Response{"changeId", strconv.FormatUint(selfId, 10)}
		jsonbyte2, err3 := json.Marshal(response2)
		if err3 != nil {
			zlog.Error(err3)
		}

		err4 := request.GetConnection().SendMessage(request.GetMsgType(), jsonbyte2)
		if err4 != nil {
			zlog.Error(err4)
		}*/

		server.GWServer.GetConnMgr().JoinRoom(request.GetConnection().GetConnID(), hostId)

		//conn.SendMessage(4, []byte("RequestLobbyData"))
		return
	}

	//如果上面没有发送成功请求则 加入请求失败
	if conn != nil {
		conn.SendMessage(4, []byte("Fail to join"))
	} else {
		zlog.Error("conn is nil.")
	}

}

func main() {
	initCmd()
	ConfigLocalFilesystemLogger("./", "face-service.log", time.Duration(86400)*time.Second, time.Duration(604800)*time.Second)
	var err error = nil
	bindAddress := ""
	if err = configs.LoadConfig(configFile); err != nil {
		fmt.Println("Load config json error:", err)
	}
	//暂时不需要redis数据库
	// common.InitRedis()
	server.GWServer = server.NewServer()

	//配置路由
	//server.GWServer.AddRouter("ping", &PingRouter{})
	//server.GWServer.AddRouter("Upload", &TD2拼接Router{})
	server.GWServer.AddRouter("Query", &QueryRouter{})
	server.GWServer.AddRouter("Join", &JoinRouter{})
	server.GWServer.AddRouter("CreateRoom", &server.CreateRoomRouter{})
	server.GWServer.AddRouter("SyncPlayer", &server.PlayerInfoRouter{})
	bindAddress = fmt.Sprintf("%s:%d", configs.GConf.Ip, configs.GConf.Port)
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.GET("/", server.WsHandler)
	r.Run(bindAddress)
}
