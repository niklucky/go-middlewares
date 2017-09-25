package middlewares

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

/*
WebsocketClient - client that listens for events and sends actions to Websocket server
*/
type WebsocketClient struct {
	Host
	Conn         *websocket.Conn
	dataHandler  func(interface{})
	errorHandler func(interface{})
}

/*
OnData - handler will handle incoming data
*/
func (ws *WebsocketClient) OnData(h func(interface{})) {
	ws.dataHandler = h
}

/*
OnError - handler will handle incoming data
*/
func (ws *WebsocketClient) OnError(h func(interface{})) {
	ws.errorHandler = h
}

/*
Connect - connecting to WS server
*/
func (ws *WebsocketClient) Connect() error {
	flag.Parse()
	log.SetFlags(0)

	url := ws.getAddress()
	log.Printf("[INFO] Connecting to %s", url)

	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	ws.Conn = c
	if err != nil {
		return err
	}
	return nil
}

/*
Listen — starting to listen to WS server
*/
func (ws *WebsocketClient) Listen() {
	if ws.Conn == nil {
		ws.Connect()
	}
	go func() {
		defer ws.Conn.Close()
		for {
			var message interface{}
			err := ws.Conn.ReadJSON(&message)
			fmt.Println("New message: ", time.Now())
			if err != nil {
				go ws.handleError(err)
				return
			}
			go ws.handleData(message)
		}
	}()
}

func (ws *WebsocketClient) handleData(data interface{}) {
	if ws.dataHandler != nil {
		ws.dataHandler(data)
	}
}
func (ws *WebsocketClient) handleError(err interface{}) {
	log.Println("Read error:", err)
	if ws.errorHandler != nil {
		ws.errorHandler(err)
	}
}

/*
Send - sending action to WS server. May return error
*/
func (ws *WebsocketClient) Send(data interface{}) error {
	req, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return ws.Conn.WriteMessage(websocket.TextMessage, req)
}

func (ws *WebsocketClient) getAddress() string {
	return "ws://" + ws.Host.Host + ":" + strconv.Itoa(ws.Host.Port) + ws.Host.Path
}

/*
Close - closing connection
*/
func (ws *WebsocketClient) Close() error {
	fmt.Println("[WS][CLIENT] Closing connection: ", ws.Conn)
	if ws.Conn != nil {
		return ws.Conn.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		)
	}
	return nil
}
