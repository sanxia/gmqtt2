MQTT for Golang Client

Example gmqtt client
-------

import (

    "github.com/sanxia/gmqtt2"

)

    const TOPIC1 = "sanxia/classroom1"

    const TOPIC1 = "sanxia/classroom2"

    client := gmqtt2.NewClient("127.0.0.1", 1883)

    client.SetCliendId("gmqtt-sanxia")

    client.SetReconnInterval(5)

    client.SetMessageHandler(messageHandler)

    client.Subscribe(TOPIC1, 0)

    client.Subscribe(TOPIC2, 0)

    client.Connect()

    client.Publish(TOPIC1, "MQTT message 1", 0, false)
    client.Publish(TOPIC2, "MQTT message 2", 0, false)


    messageHandler := func(client *gmqtt2.MqttClient, message *gmqtt2.Message) {

        statusInfo := client.StatusInfo()

        fmt.Fprint(os.Stderr, "\r\n")

        fmt.Fprintf(os.Stderr, "MQTT Status %s\r\n", statusInfo)

        fmt.Fprintf(os.Stderr, "MQTT Message topic:%s, retain:%v, payload:%v\r\n", message.TopicName, message.Header.Retain, string(message.Payload))

        if client.Status.SendCount > 5 {

            client.Disconnect()

        } else {

            time.Sleep(3 * time.Second)

            msg := fmt.Sprintf("%v - %s", time.Now(), "happy new year 2018")

            client.Publish(TOPIC, msg, 0, false)
            
        }
    }

