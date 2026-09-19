package controller

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	ssservice "x-ui/web/service/shadowsocks"
)

func TestGenerateShadowsocks2022KeyAcceptsJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	controller := &InboundController{}
	engine.POST("/key", controller.generateShadowsocksKey)

	body := `{"method":"2022-blake3-aes-128-gcm"}`
	request := httptest.NewRequest(http.MethodPost, "/key", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", response.Code)
	}
	result := struct {
		Success bool   `json:"success"`
		Msg     string `json:"msg"`
		Obj     struct {
			Method   string `json:"method"`
			Password string `json:"password"`
		} `json:"obj"`
	}{}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !result.Success || result.Obj.Method != ssservice.Method2022Blake3AES128GCM {
		t.Fatalf("unexpected response: %s", response.Body.String())
	}
	decoded, err := base64.StdEncoding.DecodeString(result.Obj.Password)
	if err != nil || len(decoded) != 16 {
		t.Fatalf("unexpected generated key: length=%d err=%v", len(decoded), err)
	}
}

func TestGenerateShadowsocks2022KeyRejectsLegacyMethod(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.POST("/key", (&InboundController{}).generateShadowsocksKey)
	request := httptest.NewRequest(http.MethodPost, "/key", strings.NewReader("method=aes-256-gcm"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)

	result := struct {
		Success bool   `json:"success"`
		Msg     string `json:"msg"`
	}{}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Success || !strings.Contains(result.Msg, "不是受支持") {
		t.Fatalf("unexpected response: %s", response.Body.String())
	}
}
