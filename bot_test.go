package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBotClientSendText(t *testing.T) {
	var logger *slog.Logger
	if testing.Verbose() {
		logger = slog.Default()
	} else {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	var mux http.ServeMux
	mux.HandleFunc("POST /open-apis/bot/v2/hook/{token}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		token := r.PathValue("token")
		switch token {
		case "":
			w.Write([]byte(`{ "code": 400, "data": {}, "msg": "token is empty" }`))
		default:
			w.Write([]byte(`{ "StatusCode": 0, "StatusMessage": "success", "code": 0, "data": {}, "msg": "success" }`))
		}
	})
	s := httptest.NewServer(&mux)
	t.Cleanup(s.Close)

	testCases := []struct {
		name   string          // 测试项目
		client BotClient       // 客户端
		ctx    context.Context // ctx 对象
		err    error           // 预期错误
	}{
		{
			name: "normal",
			client: BotClient{
				Client:  s.Client(),
				Logger:  logger,
				BaseURL: s.URL,
				Token:   "85d09ddb-5937-46e7-8628-d7959a93e3af",
			},
			ctx: context.Background(),
			err: nil,
		},
		{
			name: "token empty",
			client: BotClient{
				Client:  s.Client(),
				Logger:  logger,
				BaseURL: s.URL,
				Token:   "",
			},
			ctx: context.Background(),
			err: ErrNeedToken,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.client.SendText(tc.ctx, "警告: 测试")
			if !errors.Is(err, tc.err) {
				t.Fatalf("expect %v, got %v", tc.err, err)
			}
		})
	}
}
