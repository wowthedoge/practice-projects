package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
)

type client struct {
	conn   net.Conn
	action string
}

var (
	clients  = make(chan client)
	messages = make(chan string)
)

func main() {
	go broadcaster()

	listener, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatal("Failed to start server")
	}
	defer listener.Close()

	for {
		c, _ := listener.Accept()
		go handleConn(c)
	}
}

func broadcaster() {

	clientMap := make(map[net.Conn]bool)

	for {
		select {
		case msg := <-messages:
			for c := range clientMap {
				fmt.Fprintln(c, msg)
			}
		case client := <-clients:
			if client.action == "join" {
				clientMap[client.conn] = true
			} else {
				delete(clientMap, client.conn)
			}
		}
	}
}

func handleConn(c net.Conn) {
	defer c.Close()

	clients <- client{conn: c, action: "join"}
	defer func() {
		clients <- client{conn: c, action: "leave"}
	}()

	scanner := bufio.NewScanner(c)
	for scanner.Scan() {
		messages <- scanner.Text()
	}
}
