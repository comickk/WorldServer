package main

import (
	"fmt"
	"bytes"
	"log"
	"net/http"
	"time"
	"encoding/json"
	// "strconv"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,

	CheckOrigin: func(r *http.Request) bool { return true },
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	id int

	x int
	y int
	// Buffered channel of outbound messages.
	send chan []byte
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		
		fmt.Println("跑了一个B")
		msg := make(map[string]interface{})
		msg["title"] ="quit";
		msg["who"] = c.id 			
		send2all, _ := json.Marshal(&msg)
		c.hub.broadcast <- send2all

		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))

		msg := make(map[string]interface{})
		err = json.Unmarshal(message, &msg)
		if err != nil {
			//logs.Error("Unmarshal json error:%s", err.Error())
			continue
		}

		fmt.Println(string(message))
		if title, ok := msg["title"]; ok {

			switch title {
			case "hello?":
				fmt.Println("新来一个B")
				//先向自己发送自己id
				msg["title"] = "hello" 
				msg["id"] = c.id 
				newplayer, _ := json.Marshal(&msg)
				c.send <- newplayer  //发送新来的id,确保客户端先生成自己
				fmt.Println(string(newplayer))

				for client:= range c.hub.clients {					

					if(client.id != c.id){
						//向已加入的发送新来的id			
						client.send <- newplayer					

						//向新来的发送已在线的id
						msg2 := make(map[string]interface{})
						msg2["title"] ="hello";
						msg2["id"] = client.id 
						msg2["x"] = client.x
						msg2["y"] = client.y
						oldplayer, _ := json.Marshal(&msg2)
						c.send <- oldplayer  
						fmt.Println(string(oldplayer))
					}		
				}				

				//c.hub.broadcast <- []byte(msg1)		
				//c.hub.broadcast <- message
			case "move?":				
				msg["title"] = "move"			
				if  c.id == int(msg["who"].(float64)) {
					c.x = int(msg["x"].(float64))
					c.y = int(msg["y"].(float64))	
				}
				sendall, _ := json.Marshal(&msg)				
				c.hub.broadcast <- sendall
			default:
				c.hub.broadcast <- message
			}
		}	
		// jsonStu, err := json.Marshal(message)
		// if err != nil {
		// 	fmt.Println("生成json字符串错误")
		// }
		//jsonStu是[]byte类型，转化成string类型便于查看
		//fmt.Println(string(message))
	}
}

//发送chanal
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// serveWs handles websocket requests from the peer.
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)		
		return
	}
	
	client := &Client{hub: hub, id: hub.getId()+1 , x : 0, y : 0 , conn: conn, send: make(chan []byte, 256)}
	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
}