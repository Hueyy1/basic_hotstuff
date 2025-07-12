package service

//
//type MessageValidateService struct {
//}
//
//// NewMessageValidateService creates a new MessageValidateService.
//func NewMessageValidateService() *MessageValidateService {
//	return &MessageValidateService{}
//}
//
//func (m *MessageValidateService) MatchingMsg(msg *basichotstuffpb.BasicMessage, t basichotstuffpb.BasicMessageType, viewNumber types.View) bool {
//	// curView start from 1, 2, 3
//	// we assume special new-view messages from view 0
//	return msg.GetType() == t && msg.GetViewNumber() == uint64(viewNumber)
//}
//
//var MsgValidate = NewMessageValidateService()
