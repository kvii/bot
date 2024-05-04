package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
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
			w.Write([]byte(`{ "code": -1, "data": {}, "msg": "token is empty" }`))
		case "bad_request":
			w.Write([]byte(`{ "code": 9499, "msg": "Bad Request", "data": {} }`))
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
		msg    string          // 信息
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
			msg: "测试",
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
			msg: "测试",
			err: ErrNeedToken,
		},
		{
			name: "bad request",
			client: BotClient{
				Client:  s.Client(),
				Logger:  logger,
				BaseURL: s.URL,
				Token:   "bad_request",
			},
			ctx: context.Background(),
			msg: "测试",
			err: ErrContains("Bad Request"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.client.SendText(tc.ctx, tc.msg)
			if !errors.Is(tc.err, err) {
				t.Fatalf("expect %v, got %v", tc.err, err)
			}
		})
	}
}

// ErrContains 返回一个用于判断错误信息是否包含指定字符串的错误对象
func ErrContains(s string) error {
	return contains{s}
}

type contains struct{ string }

func (e contains) Error() string {
	return fmt.Sprintf("err should contains %q", e.string)
}

func (e contains) Is(err error) bool {
	return err != nil && strings.Contains(err.Error(), e.string)
}
