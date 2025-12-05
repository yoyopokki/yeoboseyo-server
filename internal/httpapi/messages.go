package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/websocket"
)

type SendMessageRequest struct {
	FromUserID int64  `json:"from_user_id"`
	ToUserID   int64  `json:"to_user_id"`
	Content    string `json:"content"`
}

// Заглушка хранения сообщений: на практике нужно писать в БД.
func SendMessageHandler(w http.ResponseWriter, r *http.Request) {
	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	// Здесь могли бы сохранить в БД и/или отправить через WebSocket онлайн-пользователю.
	w.WriteHeader(http.StatusAccepted)
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Простейший WebSocket для личных сообщений (без роутинга и комнат)
func MessagesWebSocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "failed to upgrade", http.StatusBadRequest)
		return
	}
	defer conn.Close()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}
		// Эхо-сервер: в реальном коде нужен брокер/room manager, фильтрация по пользователям
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}


