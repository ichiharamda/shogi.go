package main

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WebSocket用のアップグレーダー
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// 盤面の状態を保持する
var boardState = NewBoard()

// 接続中のクライアントを保持するマップ
var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan Message)
var mutex = &sync.Mutex{}

type Message struct {
	Player string `json:"player"`
	Move   string `json:"move"`
	Chat   string `json:"chat"`
}

// 将棋の初期化
func NewBoard() []string {
	return []string{
		"R", "N", "B", "K", "Q", "B", "N", "R",
		"P", "P", "P", "P", "P", "P", "P", "P",
		"", "", "", "", "", "", "", "",
		"", "", "", "", "", "", "", "",
		"p", "p", "p", "p", "p", "p", "p", "p",
		"r", "n", "b", "q", "k", "b", "n", "r",
	}
}

func main() {
	r := gin.Default()

	// WebSocketのハンドラ
	r.GET("/ws", func(c *gin.Context) {
		wsHandler(c.Writer, c.Request)
	})

	// チャットや動きの通知を受け取る
	go handleMessages()

	// ホームページの表示
	r.LoadHTMLFiles("index.html")
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	r.Run(":8080")
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "WebSocket upgrade failed", http.StatusInternalServerError)
		return
	}

	defer conn.Close()
	clients[conn] = true

	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			delete(clients, conn)
			break
		}
		broadcast <- msg
	}
}

// メッセージのハンドリング
func handleMessages() {
	for {
		msg := <-broadcast
		mutex.Lock()
		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
				client.Close()
				delete(clients, client)
			}
		}
		mutex.Unlock()
	}
}
