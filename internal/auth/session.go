package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"time"
)

// Session estructura que guardará los datos cifrados en la cookie
type Session struct {
	UserID    int64     `json:"user_id"`
	Email     string    `json:"email"`
	Rol       string    `json:"rol"`
	CreatedAt time.Time `json:"created_at"`
}

const CookieName = "socios_session"

var secretKey []byte

func init() {
	// Intentar obtener la clave secreta de variable de entorno
	keyEnv := os.Getenv("SESSION_SECRET")
	if len(keyEnv) >= 32 {
		secretKey = []byte(keyEnv[:32])
		return
	}

	// Intentar leer de archivo local .session_key
	data, err := os.ReadFile(".session_key")
	if err == nil && len(data) >= 32 {
		secretKey = data[:32]
		return
	}

	// Si no existe, generamos una clave segura y la persistimos localmente
	newKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, newKey); err == nil {
		secretKey = newKey
		_ = os.WriteFile(".session_key", newKey, 0600)
		return
	}

	// Fallback de memoria en caso extremo
	secretKey = make([]byte, 32)
	_, _ = io.ReadFull(rand.Reader, secretKey)
}

// EncryptSession serializa y cifra los datos de sesión en un string codificado en base64
func EncryptSession(session *Session) (string, error) {
	plainText, err := json.Marshal(session)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(secretKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	cipherText := gcm.Seal(nonce, nonce, plainText, nil)
	return base64.URLEncoding.EncodeToString(cipherText), nil
}

// DecryptSession decodifica y descifra la cookie obteniendo la estructura Session
func DecryptSession(cookieValue string) (*Session, error) {
	cipherText, err := base64.URLEncoding.DecodeString(cookieValue)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(secretKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(cipherText) < nonceSize {
		return nil, errors.New("texto cifrado inválido")
	}

	nonce, ciphertext := cipherText[:nonceSize], cipherText[nonceSize:]
	plainText, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	var session Session
	if err := json.Unmarshal(plainText, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

// SetSessionCookie cifra y guarda la sesión en una cookie HTTP-only y segura
func SetSessionCookie(w http.ResponseWriter, session *Session) error {
	val, err := EncryptSession(session)
	if err != nil {
		return err
	}

	cookie := &http.Cookie{
		Name:     CookieName,
		Value:    val,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Secure:   false, // Cambiar a true si se ejecuta bajo HTTPS en producción
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
	return nil
}

// ClearSessionCookie borra la cookie del navegador
func ClearSessionCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
	}
	http.SetCookie(w, cookie)
}
