package server

import (
	"errors"
	"fmt"
	"strconv"
	"sync"
	"wsserver/iserverface"
	"wsserver/zlog"
)

/*
	连接管理模块
*/
type ConnManager struct {
	connections map[uint64]iserverface.IConnection //管理的连接信息
	connLock    sync.RWMutex                       //读写连接的读写锁

	roomLock sync.RWMutex                         //读写房间映射的读写锁
	rooms    map[uint64][]iserverface.IConnection // 房间映射

	idLock   sync.RWMutex
	idToRoom map[uint64]uint64 // 客户端id 所在房间hostId的映射
}

/*
	创建一个链接管理
*/
func NewConnManager() *ConnManager {
	return &ConnManager{
		connections: make(map[uint64]iserverface.IConnection),
		rooms:       make(map[uint64][]iserverface.IConnection),
		idToRoom:    make(map[uint64]uint64),
		//rooms:       make(map[uint64][]iserverface.IConnection),
	}
}

//添加链接
func (connMgr *ConnManager) Add(conn iserverface.IConnection) {
	//保护共享资源Map 加写锁
	connMgr.connLock.Lock()
	defer connMgr.connLock.Unlock()

	//将conn连接添加到ConnMananger中
	connMgr.connections[conn.GetConnID()] = conn

	//通知一下 加入者 修改id
	err4 := conn.SendStringAsMessage("changeId", strconv.FormatUint(conn.GetConnID(), 10))
	if err4 != nil {
		zlog.Error(err4)
	}

	fmt.Println("connection add to ConnManager successfully: conn num = ", connMgr.Len())

	//两个玩家开始帧同步的测试
	/*if connMgr.Len() == 2 {
		RoomState_locked.Lock()
		RoomState_locked.M[0] = true
		RoomState_locked.Unlock()
		stop := make(chan bool)
		go SendLogicFrames(stop)
	}*/
}

//删除连接
func (connMgr *ConnManager) Remove(conn iserverface.IConnection) {
	//保护共享资源Map 加写锁
	connMgr.connLock.Lock()
	defer connMgr.connLock.Unlock()

	//删除连接信息
	delete(connMgr.connections, conn.GetConnID())
	fmt.Println("connection Remove ConnID=", conn.GetConnID(), " successfully: conn num = ", connMgr.Len())

	//断开连接也需要从房间退出来 //注意直接调用 可能导致死锁
	connMgr.QuitRoom(conn.GetConnID())
}

//利用ConnID获取链接
func (connMgr *ConnManager) Get(connID uint64) (iserverface.IConnection, error) {
	//保护共享资源Map 加读锁
	connMgr.connLock.RLock()
	defer connMgr.connLock.RUnlock()

	if conn, ok := connMgr.connections[connID]; ok {
		return conn, nil
	} else {
		return nil, errors.New("connection not found")
	}
}

//获得所有的host id
func (connMgr *ConnManager) GetAllHostIds() []uint64 {
	//保护共享资源Map 加读锁
	connMgr.connLock.RLock()
	defer connMgr.connLock.RUnlock()

	var keys []uint64

	for _, key := range connMgr.connections {
		keys = append(keys, key.GetConnID())
	}
	return keys
}

//获取当前连接
func (connMgr *ConnManager) Len() int {
	return len(connMgr.connections)
}

//清除并停止所有连接
func (connMgr *ConnManager) ClearConn() {
	//保护共享资源Map 加写锁
	connMgr.connLock.Lock()
	defer connMgr.connLock.Unlock()

	//停止并删除全部的连接信息
	for connID, conn := range connMgr.connections {
		//停止
		conn.Close()
		//删除
		delete(connMgr.connections, connID)
	}

	fmt.Println("Clear All connections successfully: conn num = ", connMgr.Len())
}

func (connMgr *ConnManager) PushAll(msg []byte) {
	for _, conn := range connMgr.connections {
		conn.SendMessage(1, msg)
	}
}

func (connMgr *ConnManager) GetConnByProName(key string, val string) (iserverface.IConnection, error) {
	//保护共享资源Map 加读锁
	connMgr.connLock.RLock()
	defer connMgr.connLock.RUnlock()
	var ConnID uint64
	for connId, conn := range connMgr.connections {

		if proVal, err := conn.GetProperty("parkid"); err != nil {
			fmt.Println("获取属性出错", err)
		} else {

			if proVal == val {
				ConnID = connId
				break
			}
		}
	}
	if conn, ok := connMgr.connections[ConnID]; ok {
		return conn, nil
	} else {
		return nil, errors.New("connection not found")
	}
}

//添加房间
func (connMgr *ConnManager) AddRoom(id uint64) {
	//保护共享资源Map 加写锁
	connMgr.roomLock.Lock()
	connMgr.idLock.Lock()
	defer connMgr.roomLock.Unlock()
	defer connMgr.idLock.Unlock()

	//新建房间列表 并将 主机id 加入其中
	connMgr.rooms[id] = []iserverface.IConnection{connMgr.connections[id]}
	connMgr.idToRoom[id] = id
}

//删除房间
func (connMgr *ConnManager) RemoveRoom(id uint64, needSyncLock bool) {
	if needSyncLock {
		//保护共享资源Map 加写锁
		connMgr.roomLock.Lock()
		connMgr.idLock.Lock()
		defer connMgr.roomLock.Unlock()
		defer connMgr.idLock.Unlock()
	}
	fmt.Println("抵达")

	//把 idToroom映射中 所处房间关系删除
	//遍历数组时 第一项是index
	for _, conn := range connMgr.rooms[id] {
		delete(connMgr.idToRoom, conn.GetConnID())
	}
	fmt.Println("房间已经删除： " + string(id))
	connMgr.PushAllInRoom(id, []byte("Room deleted"))
	delete(connMgr.rooms, id)
}

//获取房间总数
func (connMgr *ConnManager) RoomLen() int {

	return len(connMgr.rooms)
}

//像房间内广播
func (connMgr *ConnManager) PushAllInRoom(id uint64, msg []byte) {
	for _, conn := range connMgr.rooms[id] {
		conn.SendMessage(1, msg)
	}
}

//房间内广播 排除exceptId
func (connMgr *ConnManager) PushAllInRoomExceptOne(id uint64, msg []byte, exceptId uint64) {
	for _, conn := range connMgr.rooms[id] {
		if conn.GetConnID() != exceptId {
			conn.SendMessage(1, msg)
		}
	}
}

//查询房间是否可用
func (connMgr *ConnManager) QueryRoomStatus(id uint64) (bool, int) {
	//保护共享资源Map 加写锁
	connMgr.roomLock.Lock()
	defer connMgr.roomLock.Unlock()

	if conns, ok := connMgr.rooms[id]; ok {
		if len(conns) >= 4 {
			return false, -1
		} else {
			return true, len(conns)
		}
	} else {
		return false, -1
	}
}

//查询所有可进入房间
func (connMgr *ConnManager) QueryRooms() []uint64 {
	//保护共享资源Map 加写锁
	connMgr.roomLock.Lock()
	defer connMgr.roomLock.Unlock()

	var result []uint64

	for key := range connMgr.rooms {
		result = append(result, key)
	}

	return result
}

//查找客户端所在房间主机id
func (connMgr *ConnManager) QueryHostId(id uint64) uint64 {
	//保护共享资源Map 加写锁
	connMgr.idLock.Lock()
	defer connMgr.idLock.Unlock()

	return connMgr.idToRoom[id]
}

//加入房间
func (connMgr *ConnManager) JoinRoom(selfId uint64, hostId uint64) {
	connMgr.idLock.Lock()
	connMgr.roomLock.Lock()
	defer connMgr.idLock.Unlock()
	defer connMgr.roomLock.Unlock()

	if len(connMgr.rooms[hostId]) >= 4 {
		zlog.Error("Room is full.")
		return
	}
	conn, _ := connMgr.Get(selfId)
	connMgr.rooms[hostId] = append(connMgr.rooms[hostId], conn)
	connMgr.idToRoom[selfId] = hostId

}

//退出房间
func (connMgr *ConnManager) QuitRoom(selfId uint64) {
	connMgr.idLock.Lock()
	connMgr.roomLock.Lock()
	defer connMgr.idLock.Unlock()
	defer connMgr.roomLock.Unlock()

	//首先找到所在房间 uint64的默认值是0 所以用0来区别 是否为nil
	var hostId = connMgr.idToRoom[selfId]
	if hostId != 0 {
		connMgr.quit(selfId, hostId)
	}
}

//退出的底层方法 实际操作中应当调用上层接口QuitRoom 或者 RemoveRoom
//在上层接口中已经加锁 所以默认不加同步锁
func (connMgr *ConnManager) quit(id uint64, hostId uint64) {
	//无论是主机还是客机退出 都是对出自己以外的所有人广播
	var BoardcastCandidates []iserverface.IConnection

	if hostId == id {
		//主机退出的场景

		for _, conn := range connMgr.rooms[hostId] {
			guestId := conn.GetConnID()
			//先把idToRoom中引用全部删除
			delete(connMgr.idToRoom, guestId)
			//顺便把 等下需要广播的对象储存到 候选人中
			if guestId != hostId {
				BoardcastCandidates = append(BoardcastCandidates, conn)
			}
		}
		//之后把rooms中hostId引用删除
		delete(connMgr.rooms, hostId)

		//最后发送主机退出的广播
		for _, conn := range BoardcastCandidates {
			conn.SendStringAsMessage("hostQuit", "")
		}
	} else {
		//处理guest退出的场景
		for _, conn := range connMgr.rooms[hostId] {
			guestId := conn.GetConnID()
			//先把idToRoom中引用删除
			if guestId == id {
				delete(connMgr.idToRoom, guestId)
			} else {
				//把广播候选人加入 候选人数组
				BoardcastCandidates = append(BoardcastCandidates, conn)
			}
		}
		//之后把rooms中hostId引用删除
		var index1 int
		for index, conn := range connMgr.rooms[hostId] {
			if conn.GetConnID() == id {
				index1 = index
				break
			}
		}
		connMgr.rooms[hostId] = append(connMgr.rooms[hostId][:index1], connMgr.rooms[hostId][index1+1:]...)
		//最后发送客机退出的广播
		for _, conn := range BoardcastCandidates {
			fmt.Println(id)
			fmt.Println(hostId)
			conn.SendStringAsMessage("guestQuit", strconv.FormatUint(id, 10))
		}
	}
}
