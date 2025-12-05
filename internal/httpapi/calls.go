package httpapi

import (
	"encoding/json"
	"net/http"
)

// Простейший сигнальный слой для WebRTC.
// Клиенты обмениваются offer/answer/ICE через этот backend (обычно по userID/roomID).

type CallSignal struct {
	FromUserID int64           `json:"from_user_id"`
	ToUserID   int64           `json:"to_user_id"`
	Payload    json.RawMessage `json:"payload"` // SDP или ICE candidate
}

func CallOfferHandler(w http.ResponseWriter, r *http.Request) {
	handleSignal(w, r)
}

func CallAnswerHandler(w http.ResponseWriter, r *http.Request) {
	handleSignal(w, r)
}

func CallCandidateHandler(w http.ResponseWriter, r *http.Request) {
	handleSignal(w, r)
}

// Заглушка: просто принимает JSON и отдаёт 202.
// В реальном проекте тут логика доставки сигнала второму участнику (через WS/очередь/room manager).
func handleSignal(w http.ResponseWriter, r *http.Request) {
	var sig CallSignal
	if err := json.NewDecoder(r.Body).Decode(&sig); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}


