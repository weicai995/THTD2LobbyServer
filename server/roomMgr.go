package server

import (
	"sync"
	"wsserver/iserverface"
)

/*
	连接管理模块
*/
type RoomManager struct {
	roomLock sync.RWMutex                         //读写房间映射的读写锁
	rooms    map[uint64][]iserverface.IConnection // 房间映射
}

/*
	创建一个链接管理
*/
func NewRoomManager() *RoomManager {
	return &RoomManager{
		rooms: make(map[uint64][]iserverface.IConnection),
		//rooms:       make(map[uint64][]iserverface.IConnection),
	}
}

/*
//添加房间
func (roomMgr *RoomManager) Add(id uint64) {
	//保护共享资源Map 加写锁
	roomMgr.roomLock.Lock()
	defer roomMgr.roomLock.Unlock()

	//新建房间列表 并将 主机id 加入其中
	roomMgr.rooms[id] = []iserverface.IConnection{(id)}
}*/

//删除房间
func (roomMgr *RoomManager) Remove(id uint64) {
	//保护共享资源Map 加写锁
	roomMgr.roomLock.Lock()
	defer roomMgr.roomLock.Unlock()

	delete(roomMgr.rooms, id)
}

//获取房间总数
func (roomMgr *RoomManager) Len() int {

	return len(roomMgr.rooms)
}

/*//获取房间总数
func (roomMgr *RoomManager) ClearRoom() {
	//保护共享资源Map 加写锁
	roomMgr.roomLock.Lock()
	defer roomMgr.roomLock.Unlock()

	//停止并删除全部的连接信息
	for connID, conn := range connMgr.connections {
		//停止
		conn.Close()
		//删除
		delete(connMgr.connections, connID)
	}

}
*/
