package server

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"net/http"
	"sync"
	"wsserver/configs"
	"wsserver/iserverface"
	"wsserver/zlog"
)

type Server struct {
	//服务器名称
	Name string
	//IPversion
	IPversion string
	//服务器IP地址
	IP string
	//端口号
	Port int
	//Server的消息管理模块
	MsgHandler iserverface.IMsgHandle
	//当前Server链接管理器
	ConnMgr iserverface.IConnMgr
	//当前Server连接创建时的hook函数
	OnConnStart func(conn iserverface.IConnection)
	//当前Server连接断开时的hook函数
	OnConnStop func(conn iserverface.IConnection)
}

var (
	GWServer iserverface.IServer
)

func NewServer() iserverface.IServer {
	return &Server{
		Name:       configs.GConf.Name,
		IPversion:  configs.GConf.IpVersion,
		IP:         configs.GConf.Ip,
		Port:       configs.GConf.Port,
		ConnMgr:    NewConnManager(),
		MsgHandler: NewMsgHandle(),
	}
}

func (s *Server) Start(c *gin.Context) {
	fmt.Printf("[START] Server name: %s,listenner at IP: %s, Port %d is starting\n", s.Name, s.IP, s.Port)

	// Create a new Node with a Node number of 1
	node, err := snowflake.NewNode(1)
	if err != nil {
		fmt.Println(err)
		return
	}

	//开启一个go去做服务端Linster业务
	go func() {
		/*
			//TODO server.go 应该有一个自动生成ID的方法
			curConnId := uint64(time.Now().Unix())
			connId := atomic.AddUint64(&curConnId, 1)*/

		//使用雪花ID
		connId := uint64(node.Generate())
		//3.1 阻塞等待客户端建立连接请求
		var (
			err        error
			wsSocket   *websocket.Conn
			wsUpgrader = websocket.Upgrader{
				// 允许所有CORS跨域请求
				CheckOrigin: func(r *http.Request) bool {
					return true
				},
			}
		)

		if wsSocket, err = wsUpgrader.Upgrade(c.Writer, c.Request, nil); err != nil {
			return
		}
		fmt.Println("Get conn remote addr = ", wsSocket.RemoteAddr().String())
		//3 启动server网络连接业务

		//3.2 设置服务器最大连接控制,如果超过最大连接，那么则关闭此新的连接
		/*
			if s.ConnMgr.Len() >= configs.GConf.MaxConn {
				wsSocket.Close()
				continue
			}
			**/
		//3.3 处理该新连接请求的 业务 方法， 此时应该有 handler 和 conn是绑定的
		dealConn := NewConnection(s, wsSocket, connId, s.MsgHandler)

		fmt.Println("Current connId:", connId)
		//3.4 启动当前链接的处理业务
		go dealConn.Start()

	}()
}

//停止服务
func (s *Server) Stop() {
	fmt.Println("[STOP] Websocket server , name ", s.Name)

	//将其他需要清理的连接信息或者其他信息 也要一并停止或者清理
	s.ConnMgr.ClearConn()
}

//运行服务
func (s *Server) Serve(c *gin.Context) {
	s.Start(c)

	//TODO Server.Serve() 是否在启动服务的时候 还要处理其他的事情呢 可以在这里添加

	//阻塞,否则主Go退出， listenner的go将会退出
	select {}
}

//路由功能：给当前服务注册一个路由业务方法，供客户端链接处理使用
func (s *Server) AddRouter(msgId string, router iserverface.IRouter) {
	s.MsgHandler.AddRouter(msgId, router)
}

//得到链接管理
func (s *Server) GetConnMgr() iserverface.IConnMgr {
	return s.ConnMgr
}

//设置该Server的连接创建时Hook函数
func (s *Server) SetOnConnStart(hookFunc func(iserverface.IConnection)) {
	s.OnConnStart = hookFunc
}

//设置该Server的连接断开时的Hook函数
func (s *Server) SetOnConnStop(hookFunc func(iserverface.IConnection)) {
	s.OnConnStop = hookFunc
}

//调用连接OnConnStart Hook函数
func (s *Server) CallOnConnStart(conn iserverface.IConnection) {
	if s.OnConnStart != nil {
		fmt.Println("---> CallOnConnStart....")
		s.OnConnStart(conn)
	}
}

//调用连接OnConnStop Hook函数
func (s *Server) CallOnConnStop(conn iserverface.IConnection) {
	if s.OnConnStop != nil {
		fmt.Println("---> CallOnConnStop....")
		s.OnConnStop(conn)
	}
}

var RoomState_locked = struct {
	sync.RWMutex
	M map[int]bool
}{M: make(map[int]bool)}
var NextMessage = make([]byte, 0)

// 消息缓存的映射 key是房间号， value是此房间使用的消息缓存
var MessageCacheMap = struct {
	sync.RWMutex
	M map[int][]byte
}{M: make(map[int][]byte)}

//帧同步逻辑帧更新
//// 每33ms 像所有客户端发送一次 逻辑帧
//func SendLogicFrames(stop chan bool) {
//	//在房间状态开始前 先阻塞收发消息
//	/*for {
//		roomState_locked.RLock()
//		rs := roomState_locked.m[0]
//		roomState_locked.RUnlock()
//		if rs {
//			break
//		}
//	}
//	if len(NextMessage) > 0 {
//		//先用 广播来做 TODO 不能广播给别的房间的客户端
//		GWServer.GetConnMgr().PushAll(NextMessage)
//		NextMessage = make([]byte, 0)
//	} else {
//		// 当没有任何用户输入时 给所有客户端发送一个 长度为0 的消息
//		//先用 广播来做 TODO 不能广播给别的房间的客户端
//		GWServer.GetConnMgr().PushAll([]byte{0, 0, 0, 0, 0})
//	}*/
//
//	//time.AfterFunc(33*time.Millisecond, SendLogicFrames)
//
//	for {
//		RoomState_locked.RLock()
//		rs := RoomState_locked.M[0]
//		RoomState_locked.RUnlock()
//		if !rs {
//			continue
//		}
//		select {
//		case <-stop:
//			fmt.Println("Stop Routine for 逻辑帧更新..")
//			return
//		default:
//			if len(NextMessage) > 0 {
//				//先用 广播来做 TODO 不能广播给别的房间的客户端
//				GWServer.GetConnMgr().PushAll(NextMessage)
//				NextMessage = make([]byte, 0)
//			} else {
//				// 当没有任何用户输入时 给所有客户端发送一个 长度为0 的消息
//				//先用 广播来做 TODO 不能广播给别的房间的客户端
//				//fmt.Println("发了一个空包")
//				GWServer.GetConnMgr().PushAll([]byte{0, 0, 0, 0, 0})
//			}
//			time.Sleep(17 * time.Millisecond)
//		}
//	}
//}

// 处理创建房间请求
type CreateRoomRouter struct {
	BaseRouter
}

func (this *CreateRoomRouter) Handle(request iserverface.IRequest) {

	/*	conn, err := GWServer.GetConnMgr().Get(hostId)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(conn)
	*/
	GWServer.GetConnMgr().AddRoom(request.GetConnection().GetConnID())
}

// 处理 收到玩家位置数据的回复 通知房间内所有客户端更新
type PlayerInfoRouter struct {
	BaseRouter
}

func (this *PlayerInfoRouter) Handle(request iserverface.IRequest) {
	//先解析 json
	var m JsonRequest
	err2 := json.Unmarshal(request.GetData(), &m) //转到通用接口中
	if err2 != nil {
		log.Fatal(err2)
	}
	selfId := request.GetConnection().GetConnID()
	//找到所在房间id
	hostid := GWServer.GetConnMgr().QueryHostId(selfId)

	//不需要给自己发送消息
	b, _ := json.Marshal(m.Params[0])
	response := Response{"SyncPlayer", string(b)}
	jsonbyte, err3 := json.Marshal(response)
	if err3 != nil {
		zlog.Error(err3)
	}
	GWServer.GetConnMgr().PushAllInRoomExceptOne(hostid, jsonbyte, selfId)

}

func Int32ToBytes(n int) []byte {
	//int64可能无法转换成int32
	data := int32(n)
	bytebuf := bytes.NewBuffer([]byte{})
	//得注意大小端
	binary.Write(bytebuf, binary.LittleEndian, data)
	return bytebuf.Bytes()
}

type JsonRequest struct {
	Params  []interface{}
	msgType string
}
