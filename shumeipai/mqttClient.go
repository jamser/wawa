// Copyright 2014 hey Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	"fmt"
	"strings"
	_ "time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

//MqttClient mqtt客户端
type MqttClient struct {
	client       MQTT.Client
	waitingQueue map[string]func(client MQTT.Client, msg MQTT.Message)
	currID       int64
}

//GetDefaultOptions 获取默认设置
func (mqtt *MqttClient) GetDefaultOptions(addrURI string) *MQTT.ClientOptions {
	mqtt.currID = 0
	mqtt.waitingQueue = make(map[string]func(client MQTT.Client, msg MQTT.Message))
	opts := MQTT.NewClientOptions()
	opts.AddBroker(addrURI)
	opts.SetClientID("1")
	opts.SetUsername("")
	opts.SetPassword("")
	opts.SetCleanSession(false)
	opts.SetProtocolVersion(3)
	opts.SetAutoReconnect(false)
	opts.SetDefaultPublishHandler(func(client MQTT.Client, msg MQTT.Message) {
		//收到消息
		if callback, ok := mqtt.waitingQueue[msg.Topic()]; ok {
			//有等待消息的callback 还缺一个信息超时的处理机制
			ts := strings.Split(msg.Topic(), "/")
			if len(ts) > 2 {
				//这个topic存在msgid 那么这个回调只使用一次
				delete(mqtt.waitingQueue, msg.Topic())
			}
			go callback(client, msg)
		}
	})
	return opts
}

//Connect 连接mqtt服务端
func (mqtt *MqttClient) Connect(opts *MQTT.ClientOptions) error {
	fmt.Println("Connect...")
	mqtt.client = MQTT.NewClient(opts)
	if token := mqtt.client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

//GetClient 获取client
func (mqtt *MqttClient) GetClient() MQTT.Client {
	return mqtt.client
}

//Finish 关闭客户端
func (mqtt *MqttClient) Finish() {
	mqtt.client.Disconnect(250)
}

//Request 向服务器发送一条消息
/**
 * 向服务器发送一条消息
 * @param topic
 * @param msg
 * @param callback
 */
func (mqtt *MqttClient) Request(topic string, body []byte) (MQTT.Message, error) {
	mqtt.currID = mqtt.currID + 1
	topic = fmt.Sprintf("%s/%d", topic, mqtt.currID) //给topic加一个msgid 这样服务器就会返回这次请求的结果,否则服务器不会返回结果
	result := make(chan MQTT.Message)
	mqtt.On(topic, func(client MQTT.Client, msg MQTT.Message) {
		result <- msg
	})
	mqtt.GetClient().Publish(topic, 0, false, body)
	msg, ok := <-result
	if !ok {
		return nil, fmt.Errorf("client closed")
	}
	return msg, nil
}

//RequestNR 向服务器发送一条消息，无返回
/**
 * 向服务器发送一条消息,但不要求服务器返回结果
 * @param topic
 * @param msg
 */
func (mqtt *MqttClient) RequestNR(topic string, body []byte) {
	mqtt.GetClient().Publish(topic, 0, false, body)
}

//On 监听指定类型的topic消息
/**
 * 监听指定类型的topic消息
 * @param topic
 * @param callback
 */
func (mqtt *MqttClient) On(topic string, callback func(client MQTT.Client, msg MQTT.Message)) {
	////服务器不会返回结果
	mqtt.waitingQueue[topic] = callback //添加这条消息到等待队列
}
