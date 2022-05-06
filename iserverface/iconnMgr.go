package iserverface

type IConnMgr interface {
	Add(conn IConnection)                   //添加链接
	Remove(conn IConnection)                //删除连接
	Get(connID uint64) (IConnection, error) //利用ConnID获取链接
	Len() int                               //获取当前连接
	ClearConn()                             //删除并停止所有链接
	PushAll([]byte)                         //广播
	GetAllHostIds() []uint64                //获取所有主机id

	AddRoom(id uint64)                       //添加房间
	RemoveRoom(id uint64, needSyncLock bool) //删除房间 第二个参数代表是否需要 加锁 用于防止死锁
	//Get(connID uint64) (IConnection, error) //利用ConnID获取链接
	RoomLen() int //获取当前房间总数
	//ClearRoom()               //删除并关闭所有房间
	PushAllInRoom(id uint64, msg []byte)                           //全房间广播
	PushAllInRoomExceptOne(id uint64, msg []byte, exceptId uint64) //排除exceptId的房间广播
	QueryRoomStatus(id uint64) (bool, int)                         //查询房间是否可用 连接断开或者人数已满则 不可用
	QueryRooms() []uint64                                          //查找所有可用房间
	QueryHostId(id uint64) uint64                                  // 查找客户端所在房间主机id
	JoinRoom(selfId uint64, hostId uint64)                         //加入房间的接口
	QuitRoom(selfId uint64)                                        //退出房间的接口
}
