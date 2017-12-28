package gmqtt2

import (
	"net"
)

import (
	proto "github.com/huin/mqtt"
	"github.com/jeffallen/mqtt"
)

/* ================================================================================
 * Oauth
 * qq group: 582452342
 * email   : 2091938785@qq.com
 * author  : 美丽的地球啊
 * ================================================================================ */

type QosLevel uint8

const (
	QosAtMostOnce = QosLevel(iota)
	QosAtLeastOnce
	QosExactlyOnce
)

type (
	MqttClient struct {
		username       string
		password       string
		host           string
		port           int
		clientId       string
		topics         map[string]proto.QosLevel
		messageHandler func(*MqttClient, *Message)
		conn           net.Conn
		client         *mqtt.ClientConn
		reconnInterval int
		reconnCount    int
		isReconn       bool
		isConnected    bool
		isDisconnect   bool
		connError      chan bool
		connSuccess    chan bool
		Status         *ClientStatus
	}

	//消息
	Message struct {
		Header    MessageHeader
		MessageId uint16
		TopicName string
		Payload   []byte
	}

	//消息头
	MessageHeader struct {
		QosLevel QosLevel
		DupFlag  bool
		Retain   bool
	}

	//客户端状态
	ClientStatus struct {
		SendCount   int
		ReceivCount int
		ReconnCount int
		ErrorCount  int
	}
)
