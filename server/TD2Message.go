package server

type TD2Message struct {
	Id      int
	Method  string
	Params  []string
	MsgType int
}

//创建一个Message消息包
func NewTD2Msg(msgID int, msgType int, params []string, method string) *TD2Message {
	return &TD2Message{
		Id:      msgID,
		Method:  method,
		Params:  params,
		MsgType: msgType,
	}
}

//获取消息类型
func (msg *TD2Message) GetTD2MsgType() int {
	return msg.MsgType
}

//获取消息类型
func (msg *TD2Message) GetTD2MsgID() int {
	return msg.Id
}
