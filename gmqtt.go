package gmqtt2

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"time"
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

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 初始化MqttClient
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func NewClient(host string, port int) *MqttClient {
	if port == 0 {
		port = 1883
	}

	client := &MqttClient{
		host:           host,
		port:           port,
		reconnInterval: 5,
		reconnCount:    0,
		isReconn:       true,
		isConnected:    false,
		Status: &ClientStatus{
			SendCount:   0,
			ReceivCount: 0,
		},
		topics:      make(map[string]proto.QosLevel, 0),
		connError:   make(chan bool),
		connSuccess: make(chan bool),
	}

	//监视连接状态
	client.monitorConnectStatus()

	//派发消息
	client.dispatchMessage()

	return client
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 连接服务器
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (s *MqttClient) Connect() error {
	fmt.Fprint(os.Stderr, "开始连接服务器\n")
	var err error
	var count int = 1

	//连接服务器
	for {
		err := s.dialTcp()
		if err == nil {
			break
		}

		if s.isReconn {
			//超过重连最大次数则返回
			if s.reconnCount > 0 && count >= s.reconnCount {
				break
			}

			fmt.Fprintf(os.Stderr, "第 %d 次连接尝试, err: %v\n", count, err)
			time.Sleep(time.Duration(int64(s.reconnInterval)) * time.Second)

			s.Status.ReconnCount++
			count++
		}
	}
	fmt.Fprint(os.Stderr, "连接服务器成功\n")

	s.isConnected = true

	//已成功连接
	s.connSuccess <- true

	return err
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 连接Mqtt服务器
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (s *MqttClient) dialTcp() error {
	fmt.Fprint(os.Stderr, "发起网络连接\n")

	if s.conn != nil {
		s.conn.Close()
	}

	//连接到 mqtt server
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	if conn, err := net.Dial("tcp", addr); err != nil {
		return err
	} else {
		s.conn = conn
	}

	//初始化mqtt client
	client := mqtt.NewClientConn(s.conn)
	client.Dump = true
	client.ClientId = s.clientId
	if err := client.Connect(s.username, s.password); err != nil {
		return err
	}

	//订阅
	tqs := make([]proto.TopicQos, 0)
	for topic, qos := range s.topics {
		tqs = append(tqs, proto.TopicQos{Topic: topic, Qos: qos})
	}
	client.Subscribe(tqs)

	fmt.Fprint(os.Stderr, "发起网络连接完成\r\n")
	s.client = client

	return nil
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 监视连接状态，自动重连
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (s *MqttClient) monitorConnectStatus() {
	go func() {
	FOREVER:
		for {
			select {
			case connError := <-s.connError:
				if s.isDisconnect {
					break FOREVER
				} else {
					if !s.isConnected {
						fmt.Fprintf(os.Stderr, "%s, 连接状态是否异常:%v\r\n", "监视连接状态", connError)
						if connError {
							s.Status.ErrorCount++
							fmt.Fprintf(os.Stderr, "尝试重新连接服务器\r\n")

							s.Connect()
						}
					}
				}
			}

		}
	}()
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 派发消息
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (s *MqttClient) dispatchMessage() {
	go func() {
		for {
			select {
			case connSuccess := <-s.connSuccess:
				if connSuccess {
					s.message()
				}
			}
		}
	}()
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 消息处理
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (s *MqttClient) message() {
	fmt.Fprint(os.Stderr, "派发消息\r\n")

FOREVER:
	for {
		select {
		case m := <-s.client.Incoming:
			if m == nil {
				if !s.isDisconnect {
					fmt.Fprintf(os.Stderr, "派发消息失败，准备重连\r\n")
					s.isConnected = false
					s.connError <- true
				}

				break FOREVER
			}

			if s.messageHandler != nil {
				message := &Message{
					Header: MessageHeader{
						DupFlag:  m.DupFlag,
						Retain:   m.Retain,
						QosLevel: QosLevel(m.QosLevel),
					},
				}

				data := new(bytes.Buffer)
				m.Payload.WritePayload(data)
				payload := data.Bytes()

				message.MessageId = m.MessageId
				message.TopicName = m.TopicName
				message.Payload = payload

				s.messageHandler(s, message)

				s.Status.ReceivCount++
			}
		case <-time.After(1 * time.Second):
			fmt.Fprint(os.Stderr, "Idle \r\n")
			time.Sleep(10 * time.Millisecond)
		}
	}
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 发布消息
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (s *MqttClient) Publish(topic, payload string, qosLevel QosLevel, isRetain bool) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Fprintf(os.Stderr, "发布消息网络异常，准备重连: %v\r\n", err)
			s.isConnected = false
			s.connError <- true
		}
	}()

	s.client.Publish(&proto.Publish{
		Header:    proto.Header{Retain: isRetain, QosLevel: proto.QosLevel(qosLevel)},
		TopicName: topic,
		Payload:   proto.BytesPayload([]byte(payload)),
	})

	s.Status.SendCount++
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 订阅消息
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (s *MqttClient) Subscribe(topic string, qosLevel QosLevel) {
	_, isOk := s.topics[topic]
	if !isOk {
		s.topics[topic] = proto.QosLevel(qosLevel)
	}
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 断开连接
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (s *MqttClient) Disconnect() {
	fmt.Fprintf(os.Stderr, "主动断开连接\r\n")

	s.client.Disconnect()

	close(s.connError)
	close(s.connSuccess)

	if s.conn != nil {
		s.conn.Close()
	}

	s.isConnected = false
	s.isDisconnect = true
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 设置用户名和密码
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (s *MqttClient) SetUser(username, password string) {
	s.username = username
	s.password = password
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 设置客户端Id
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (s *MqttClient) SetCliendId(clientId string) {
	s.clientId = clientId
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 设置消息处理器
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (s *MqttClient) SetMessageHandler(messageHandler func(c *MqttClient, m *Message)) {
	s.messageHandler = messageHandler
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 设置重连间隔，单位秒
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (s *MqttClient) SetReconnInterval(interval int) {
	s.reconnInterval = interval
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 设置重连最大尝试次数，0为不限制
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (s *MqttClient) SetReconnCount(count int) {
	s.reconnCount = count
}

/* ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 * 状态信息
 * ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ */
func (s *MqttClient) StatusInfo() string {
	info := fmt.Sprintf("ErrorCount:%d, ReconnCount:%d, ReceivCount:%d, SendCount:%d", s.Status.ErrorCount, s.Status.ReconnCount, s.Status.ReceivCount, s.Status.SendCount)
	return info
}
