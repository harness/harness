// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.


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
