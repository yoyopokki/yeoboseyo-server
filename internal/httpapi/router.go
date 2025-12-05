package httpapi

import (
	"net/http"

	"github.com/gorilla/mux"
)

func RegisterRoutes(r *mux.Router) {
	// Auth
	r.HandleFunc("/auth/google/login", GoogleLoginHandler).Methods(http.MethodGet)
	r.HandleFunc("/auth/google/callback", GoogleCallbackHandler).Methods(http.MethodGet)
	// POST — основной хендлер, OPTIONS — для CORS preflight
	r.HandleFunc("/auth/google/frontend", GoogleFrontendAuthHandler).Methods(http.MethodPost)
	r.HandleFunc("/auth/google/frontend", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}).Methods(http.MethodOptions)

	// Protected API (in будущем можно повесить middleware аутентификации)
	r.HandleFunc("/api/me", MeHandler).Methods(http.MethodGet)

	// Direct messages
	r.HandleFunc("/api/messages/send", SendMessageHandler).Methods(http.MethodPost)
	r.HandleFunc("/api/messages/ws", MessagesWebSocketHandler).Methods(http.MethodGet)

	// Audio/video calls signaling
	r.HandleFunc("/api/call/offer", CallOfferHandler).Methods(http.MethodPost)
	r.HandleFunc("/api/call/answer", CallAnswerHandler).Methods(http.MethodPost)
	r.HandleFunc("/api/call/candidate", CallCandidateHandler).Methods(http.MethodPost)
}


