/*
 * Copyright (C) 2019  SuperGreenLab <towelie@supergreenlab.com>
 * Author: Constantin Clauzel <constantin.clauzel@gmail.com>
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/gousb"
)

var (
	server = flag.String("mqtt_server", "tcp://node.local:1883", "The full url of the MQTT server to connect to ex: tcp://127.0.0.1:1883")

	message_chan chan string
	client       MQTT.Client
)

func start_mqtt() {
	log.Println("init_mqtt")
	message_chan = make(chan string, 10)

	connOpts := MQTT.NewClientOptions().AddBroker(*server).SetClientID("akai").SetCleanSession(true)
	tlsConfig := &tls.Config{InsecureSkipVerify: true, ClientAuth: tls.NoClientCert}
	connOpts.SetTLSConfig(tlsConfig)

	client = MQTT.NewClient(connOpts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	fmt.Printf("Connected to %s\n", *server)
	for m := range message_chan {
		client.Publish("akai", 2, true, m)
	}
}

func start_akai() {
	log.Println("init_akai")
	ctx := gousb.NewContext()
	defer ctx.Close()

	dev, err := ctx.OpenDeviceWithVIDPID(0x09e8, 0x0075)
	if err != nil {
		log.Fatalf("Could not open a device: %v", err)
	}
	defer dev.Close()

	dev.SetAutoDetach(true)
	cfg, err := dev.Config(1)
	if err != nil {
		log.Fatalf("%s.Config(1): %v", dev, err)
	}
	defer cfg.Close()

	intf, err := cfg.Interface(1, 0)
	if err != nil {
		log.Fatalf("%s.DefaultInterface(): %v", dev, err)
	}
	defer intf.Close()

	ep, err := intf.InEndpoint(1)
	if err != nil {
		log.Fatalf("%s.InEndpoint(1): %v", intf, err)
	}

	for {
		buf := make([]byte, 64)
		_, err := ep.Read(buf)

		if err != nil {
			break
		}

		cmd := buf[0]
		n := buf[2]
		if cmd == 9 { // PAD down
			vel := buf[3]
			message_chan <- fmt.Sprintf("id=%d evt=down vel=%d", n, vel)
		} else if cmd == 8 { // PAD up
			message_chan <- fmt.Sprintf("id=%d evt=up", n)
		} else if cmd == 12 { // PROG CHG
			message_chan <- fmt.Sprintf("id=%d evt=prog_change", n)
		} else if cmd == 11 {
			v := buf[3]
			message_chan <- fmt.Sprintf("id=%d evt=pot v=%d", n, v)
		} else {
			log.Println(buf)
		}
	}
}

func main() {
	go start_mqtt()
	go start_akai()

	select {}
}
