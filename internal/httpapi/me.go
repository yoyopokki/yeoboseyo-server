package httpapi

import (
	"encoding/json"
	"net/http"
)

// В дальнейшем сюда вешается middleware, которая достаёт пользователя из токена/сессии.
func MeHandler(w http.ResponseWriter, r *http.Request) {
	// Заглушка: возвращаем фиктивного пользователя
	user := map[string]any{
		"id":    1,
		"email": "user@example.com",
		"name":  "Demo User",
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(user)
}


