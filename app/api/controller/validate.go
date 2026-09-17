package controller
import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/base64"
    "io"
    "net/http"
    "os"
)
const hmacSecretKey = "GITNESS_HMAC_SECRET"
func ValidateHandler(w http.ResponseWriter, r *http.Request) {
    hmacHeader := r.Header.Get("X-Hmac-SHA256")
    if hmacHeader == "" {
        http.Error(w, "Missing X-Hmac-SHA256 header", http.StatusBadRequest)
        return
    }
    bodyBytes, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "Failed to read request body", http.StatusInternalServerError)
        return
    }
    defer r.Body.Close()
    expectedHMAC := base64.StdEncoding.EncodeToString(hmac.New(sha256.New, []byte(getSecretKey())).Sum(bodyBytes))
    if !hmac.Equal([]byte(hmacHeader), []byte(expectedHMAC)) {
        http.Error(w, "Invalid HMAC signature", http.StatusUnauthorized)
        return
    }
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"status": "success"}`))
}
func getSecretKey() string {
    return os.Getenv(hmacSecretKey)
}
