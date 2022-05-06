package iserverface

type IRonnMgr interface {
	Add(id uint64)    //添加房间
	Remove(id uint64) //删除房间
	//Get(connID uint64) (IConnection, error) //利用ConnID获取链接
	Len() int //获取当前房间总数
	//ClearRoom()               //删除并关闭所有房间
	PushAll([]byte)           //广播
	QueryRoom(id uint64) bool //查询房间是否可用 连接断开或者人数已满则 不可用
	//GetAllHostIds() []uint64                //获取所有主机id
}
