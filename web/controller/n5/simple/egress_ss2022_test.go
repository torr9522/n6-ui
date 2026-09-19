package simple

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	ssservice "x-ui/web/service/shadowsocks"
)

func TestGenerateSimpleEgressShadowsocks2022KeyAcceptsJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.POST("/key", (&EgressController{}).generateShadowsocksKey)
	body := `{"method":"2022-blake3-aes-256-gcm"}`
	request := httptest.NewRequest(http.MethodPost, "/key", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)

	result := struct {
		Success bool `json:"success"`
		Obj     struct {
			Method   string `json:"method"`
			Password string `json:"password"`
		} `json:"obj"`
	}{}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !result.Success || result.Obj.Method != ssservice.Method2022Blake3AES256GCM {
		t.Fatalf("unexpected response: %s", response.Body.String())
	}
	decoded, err := base64.StdEncoding.DecodeString(result.Obj.Password)
	if err != nil || len(decoded) != 32 {
		t.Fatalf("unexpected key length=%d err=%v", len(decoded), err)
	}
}
