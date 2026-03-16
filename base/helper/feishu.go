package helper

// {
// 	"schema": "2.0",
// 	"header": {
// 			"event_id": "887f91421f7a01d4db537339ea2fc109",
// 			"token": "rdK1UvOumyzKc0OKQpI1vhFPid8j3yaC",
// 			"create_time": "1774104964877527",
// 			"event_type": "card.action.trigger",
// 			"tenant_key": "147ae47f2620975d",
// 			"app_id": "cli_a9348075d4f8dcc7"
// 	},
// 	"event": {
// 			"operator": {
// 					"tenant_key": "147ae47f2620975d",
// 					"open_id": "ou_2a1702de477a86fa0ff3cc5513b18707",
// 					"union_id": "on_1857cde1b25611e542c530d110e5b2c4"
// 			},
// 			"token": "c-91e832c0a0a1c7e14e5a1314a799bdbd9a5b6c6f",
// 			"action": {
// 					"value": {
// 							"value1": "30m",
// 							"value2": "3h"
// 					},
// 					"tag": "select_static",
// 					"option": "2"
// 			},
// 			"host": "im_message",
// 			"context": {
// 					"open_message_id": "om_x100b54def8edb8a8c2d1b7b85620b3a",
// 					"open_chat_id": "oc_1b331aaade15126c200d94627d8aa2a0"
// 			}
// 	}
// }

type FeiShuCardTrigger struct {
	Schema string `json:"schema"`
	Header Header `json:"header"`
	Event  Event  `json:"event"`
}

type Header struct {
	Event_id    string `json:"event_id"`
	Token       string `json:"token"`
	Create_time string `json:"create_time"`
	Event_type  string `json:"event_type"`
	Tenant_key  string `json:"tenant_key"`
	App_id      string `json:"app_id"`
}

type Event struct {
	Operator EventOperator `json:"operator"`
	Token    string        `json:"token"`
	Action   EventAction   `json:"action"`
	Host     string        `json:"host"`
	Context  EventContext  `json:"context"`
}

type EventOperator struct {
	Tenant_key string `json:"tenant_key"`
	Open_id    string `json:"open_id"`
	Union_id   string `json:"union_id"`
}

type EventAction struct {
	Value  map[string]string `json:"value"`
	Tag    string            `json:"tag"`
	Option string            `json:"option"`
}

type EventContext struct {
	Open_message_id string `json:"open_message_id"`
	Open_chat_id    string `json:"open_chat_id"`
}
