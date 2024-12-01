package wx

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
	mux.HandleFunc("POST /cgi-bin/webhook/send", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		key := r.URL.Query().Get("key")
		switch key {
		case "":
			w.Write([]byte(`{"errcode":93000,"errmsg":"invalid webhook url, hint: [1716264289200120408162556], from ip: 61.156.117.29, more info at https://open.work.weixin.qq.com/devtool/query?e=93000"}`))
		case "invalid_message_type":
			w.Write([]byte(`{"errcode":40008,"errmsg":"invalid message type, hint: [1716264347466642364852488], from ip: 61.156.117.29, more info at https://open.work.weixin.qq.com/devtool/query?e=40008"}`))
		case "empty_content":
			w.Write([]byte(`{"errcode":44004,"errmsg":"empty content, hint: [1716264383294863579289845], from ip: 61.156.117.29, more info at https://open.work.weixin.qq.com/devtool/query?e=44004"}`))
		default:
			w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
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
				Key:     "7532a14a-d294-4a58-an57-6da300ecf68f",
			},
			ctx: context.Background(),
			msg: "测试",
			err: nil,
		},
		{
			name: "empty key",
			client: BotClient{
				Client:  s.Client(),
				Logger:  logger,
				BaseURL: s.URL,
				Key:     "",
			},
			ctx: context.Background(),
			msg: "测试",
			err: ErrNeedToken,
		},
		{
			name: "invalid message type",
			client: BotClient{
				Client:  s.Client(),
				Logger:  logger,
				BaseURL: s.URL,
				Key:     "invalid_message_type",
			},
			ctx: context.Background(),
			msg: "测试",
			err: ErrContains("invalid message type"),
		},
		{
			name: "empty content",
			client: BotClient{
				Client:  s.Client(),
				Logger:  logger,
				BaseURL: s.URL,
				Key:     "empty_content",
			},
			ctx: context.Background(),
			msg: "测试",
			err: ErrContains("empty content"),
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

func TestBotClient_SendMarkDown(t *testing.T) {
	var logger *slog.Logger
	if testing.Verbose() {
		logger = slog.Default()
	} else {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	var mux http.ServeMux
	mux.HandleFunc("POST /cgi-bin/webhook/send", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		key := r.URL.Query().Get("key")
		switch key {
		case "":
			w.Write([]byte(`{"errcode":93000,"errmsg":"invalid webhook url, hint: [1716264289200120408162556], from ip: 61.156.117.29, more info at https://open.work.weixin.qq.com/devtool/query?e=93000"}`))
		case "invalid_message_type":
			w.Write([]byte(`{"errcode":40008,"errmsg":"invalid message type, hint: [1716264347466642364852488], from ip: 61.156.117.29, more info at https://open.work.weixin.qq.com/devtool/query?e=40008"}`))
		case "empty_content":
			w.Write([]byte(`{"errcode":44004,"errmsg":"empty content, hint: [1716264383294863579289845], from ip: 61.156.117.29, more info at https://open.work.weixin.qq.com/devtool/query?e=44004"}`))
		default:
			w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
		}
	})
	s := httptest.NewServer(&mux)
	t.Cleanup(s.Close)

	s.URL = `https://qyapi.weixin.qq.com`

	testCasesMarkdown := []struct {
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
				Key:     "ee556a46-a3a7-4978-a186-7e3181f29da9",
			},
			ctx: context.Background(),
			msg: "1. 「11-23 9:00」[关于xxx的公告](https://example.com)\n 2.「11-23 9:00」[关于xxx的公告](https://example.com)\n ",
			err: nil,
		},
	}

	for _, tc := range testCasesMarkdown {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.client.SendMarkDown(tc.ctx, tc.msg)
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
