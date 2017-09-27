package middlewares

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
)

/*
NewRabbitMQ - RqbbitMQ middleware constructor
*/
func NewRabbitMQ(host Host, ex MQExchange) RabbitMQ {
	return RabbitMQ{
		Host:     host,
		Exchange: ex,
	}
}

/*
NewWebsocketClient - constructor for WebsocketClient
*/
func NewWebsocketClient(host Host) WebsocketClient {
	return WebsocketClient{
		Host: host,
	}
}

/*
NewWebsocketServer - Server constructor
*/
func NewWebsocketServer(host string, port int, timeout int) WebsocketServer {
	t := time.Duration(timeout) * time.Second
	return WebsocketServer{
		Host:          host,
		Port:          port,
		Timeout:       t,
		Connections:   make(map[string]*websocket.Conn),
		Subscriptions: make(map[string]map[string]time.Time),
	}
}

// Host - struct for host params config for HTTP/WS connections
type Host struct {
	Host     string
	Port     int
	User     string
	Password string
	Path     string
	Data     interface{}
}

func logOnError(err error, msg string) {
	if err != nil {
		log.Printf("%s: %s", msg, err)
	}
}
