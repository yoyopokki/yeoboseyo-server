package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"github.com/rs/zerolog/log"

	"github.com/yeoboseyo/server/internal/db"
)

var googleOAuthConfig *oauth2.Config
var jwtSecret []byte

func init() {
	googleOAuthConfig = &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"), // напр. http://localhost:8080/auth/google/callback
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	// Секрет для подписи наших JWT (для продакшена обязательно задать через env)
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-secret-change-me"
	}
	jwtSecret = []byte(secret)
}

// GoogleLoginHandler: редиректит на Google OAuth
func GoogleLoginHandler(w http.ResponseWriter, r *http.Request) {
	url := googleOAuthConfig.AuthCodeURL("state") // state в реальности нужно рандомный и хранить
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// GoogleCallbackHandler: получает код, обменивает на токен и возвращает базовый профиль
// Ищет/создаёт пользователя в БД и выдаёт свой JWT/сессию.
func GoogleCallbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}

	token, err := googleOAuthConfig.Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, "failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Получаем информацию о пользователе из Google API
	userInfo, err := fetchGoogleUserInfo(r.Context(), token.AccessToken)
	if err != nil {
		http.Error(w, "failed to fetch user info: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Ищем или создаём пользователя в БД
	pool := DB()
	if pool == nil {
		http.Error(w, "database not initialized", http.StatusInternalServerError)
		return
	}

	dbUser, isNew, err := db.GetOrCreateUser(
		r.Context(),
		pool,
		userInfo.Subject,
		userInfo.Email,
		userInfo.Name,
		userInfo.Picture,
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to get or create user")
		http.Error(w, "failed to process user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Создаём СВОЙ JWT с user_id из БД
	now := time.Now()
	claims := jwt.MapClaims{
		"user_id": dbUser.ID,
		"email":   dbUser.Email,
		"exp":     now.Add(24 * time.Hour).Unix(),
		"iat":     now.Unix(),
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := jwtToken.SignedString(jwtSecret)
	if err != nil {
		http.Error(w, "failed to sign jwt", http.StatusInternalServerError)
		return
	}

	resp := frontendGoogleAuthResponse{
		Token: signed,
		User: googleUserInfo{
			Sub:     dbUser.GoogleID,
			Email:   dbUser.Email,
			Name:    dbUser.Name,
			Picture: dbUser.Picture,
		},
	}

	if isNew {
		log.Info().Int64("user_id", dbUser.ID).Str("email", dbUser.Email).Msg("new user registered")
	} else {
		log.Info().Int64("user_id", dbUser.ID).Str("email", dbUser.Email).Msg("user logged in")
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// ==== Вариант, когда фронт уже сделал Google OAuth и прислал id_token ====

type frontendGoogleAuthRequest struct {
	IDToken string `json:"id_token"`
}

type frontendGoogleAuthResponse struct {
	Token string         `json:"token"` // наш собственный JWT
	User  googleUserInfo `json:"user"`
}

// Данные пользователя, которые мы вытащим из id_token / tokeninfo
type googleUserInfo struct {
	Sub     string `json:"sub"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

// Объект ответа от https://oauth2.googleapis.com/tokeninfo?id_token=...
type googleTokenInfo struct {
	Audience string `json:"aud"`
	Subject  string `json:"sub"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Picture  string `json:"picture"`
	Expires  int64  `json:"exp,string"`
}

// GoogleFrontendAuthHandler принимает id_token с фронта, валидирует его у Google
// и выдаёт СВОЙ JWT, который вы дальше используете в Authorization: Bearer ...
func GoogleFrontendAuthHandler(w http.ResponseWriter, r *http.Request) {
	var req frontendGoogleAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.IDToken == "" {
		http.Error(w, "invalid id_token", http.StatusBadRequest)
		return
	}

	info, err := verifyGoogleIDToken(r.Context(), req.IDToken)
	if err != nil {
		http.Error(w, "invalid google token: "+err.Error(), http.StatusUnauthorized)
		return
	}

	// Проверяем, что токен выдан для нашего клиента
	if info.Audience != googleOAuthConfig.ClientID {
		http.Error(w, "audience mismatch", http.StatusUnauthorized)
		return
	}

	// Ищем или создаём пользователя в БД
	pool := DB()
	if pool == nil {
		http.Error(w, "database not initialized", http.StatusInternalServerError)
		return
	}

	dbUser, isNew, err := db.GetOrCreateUser(
		r.Context(),
		pool,
		info.Subject,
		info.Email,
		info.Name,
		info.Picture,
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to get or create user")
		http.Error(w, "failed to process user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Создаём СВОЙ JWT с user_id из БД
	now := time.Now()
	claims := jwt.MapClaims{
		"user_id": dbUser.ID,
		"email":   dbUser.Email,
		"exp":     now.Add(24 * time.Hour).Unix(),
		"iat":     now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(jwtSecret)
	if err != nil {
		http.Error(w, "failed to sign jwt", http.StatusInternalServerError)
		return
	}

	resp := frontendGoogleAuthResponse{
		Token: signed,
		User: googleUserInfo{
			Sub:     dbUser.GoogleID,
			Email:   dbUser.Email,
			Name:    dbUser.Name,
			Picture: dbUser.Picture,
		},
	}

	if isNew {
		log.Info().Int64("user_id", dbUser.ID).Str("email", dbUser.Email).Msg("new user registered")
	} else {
		log.Info().Int64("user_id", dbUser.ID).Str("email", dbUser.Email).Msg("user logged in")
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// verifyGoogleIDToken ходит в Google tokeninfo и вытаскивает информацию о токене.
func verifyGoogleIDToken(ctx context.Context, idToken string) (*googleTokenInfo, error) {
	url := "https://oauth2.googleapis.com/tokeninfo?id_token=" + idToken
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("google tokeninfo status %d: %s", resp.StatusCode, string(body))
	}

	var info googleTokenInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}

	if info.Subject == "" {
		return nil, errors.New("missing sub in tokeninfo")
	}

	// Базовая проверка exp
	if info.Expires != 0 && time.Unix(info.Expires, 0).Before(time.Now().Add(-1*time.Minute)) {
		return nil, errors.New("token expired")
	}

	return &info, nil
}

// fetchGoogleUserInfo получает информацию о пользователе из Google API используя access_token
func fetchGoogleUserInfo(ctx context.Context, accessToken string) (*googleTokenInfo, error) {
	url := "https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + accessToken
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("google userinfo status %d: %s", resp.StatusCode, string(body))
	}

	var userInfo struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	// Преобразуем в формат googleTokenInfo для совместимости
	return &googleTokenInfo{
		Subject: userInfo.ID,
		Email:   userInfo.Email,
		Name:    userInfo.Name,
		Picture: userInfo.Picture,
	}, nil
}


