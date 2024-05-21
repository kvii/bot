package feishu

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
)

// 信息类型
type MessageType = string

const (
	MessageTypeText MessageType = "text" // 文本信息类型
)

// 信息
type Message struct {
	MsgType MessageType `json:"msg_type"` // 信息类型
	Content any         `json:"content"`  // 信息内容
}

// 文本信息
type TextMessage struct {
	Text string `json:"text"` // 文本内容
}

// 发送响应
type SendResponse[T any] struct {
	Code int    `json:"code"` // 响应码。非 0 为异常。
	Data T      `json:"data"` // 返回数据
	Msg  string `json:"msg"`  // 异常信息
}

// 预定义错误
var (
	ErrNeedToken = errors.New("feishu: need token") // 需要提供令牌
)

// 飞书机器人客户端
type BotClient struct {
	Client  *http.Client // 底层 http client。不填则使用默认值。
	Logger  *slog.Logger // 日志 logger。不填则使用默认值。
	BaseURL string       // 飞书接口基础地址。不填则使用默认值。
	Token   string       // 机器人令牌。
}

// SendText 方法发送文本信息。
// 目前只支持文本信息。信息内容需要包含指定关键字。
func (c BotClient) SendText(ctx context.Context, msg string) error {
	c.logger().InfoContext(ctx, "发送文本消息", slog.String("msg", msg))

	return c.send(ctx, Message{
		MsgType: MessageTypeText,
		Content: TextMessage{Text: msg},
	})
}

// Send 方法发送信息。
// 目前只支持文本信息。信息内容需要包含指定关键字。
func (c BotClient) Send(ctx context.Context, msg Message) error {
	c.logger().InfoContext(ctx, "发送消息", slog.String("msgType", msg.MsgType))
	return c.send(ctx, msg)
}

func (c BotClient) send(ctx context.Context, msg Message) error {
	if c.Token == "" {
		c.logger().ErrorContext(ctx, "需要提供令牌")
		return ErrNeedToken
	}

	u, err := url.Parse(c.baseURL())
	if err != nil {
		c.logger().ErrorContext(ctx, "URL 解析失败", slog.Any("err", err))
		return err
	}
	u = u.JoinPath("/open-apis/bot/v2/hook/", c.Token)

	bs, err := json.Marshal(msg)
	if err != nil {
		c.logger().ErrorContext(ctx, "参数序列化失败", slog.Any("err", err))
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewBuffer(bs))
	if err != nil {
		c.logger().ErrorContext(ctx, "请求创建失败", slog.Any("err", err))
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client().Do(req)
	if err != nil {
		c.logger().ErrorContext(ctx, "请求发送失败", slog.Any("err", err))
		return err
	}
	defer resp.Body.Close()

	bs, err = io.ReadAll(resp.Body)
	if err != nil {
		c.logger().ErrorContext(ctx, "响应读取失败", slog.Any("err", err))
		return err
	}

	if resp.StatusCode != http.StatusOK {
		c.logger().ErrorContext(ctx, "响应状态错误", slog.Int("status-code", resp.StatusCode))
		return fmt.Errorf("响应状态错误: %d", resp.StatusCode)
	}
	if mt := resp.Header.Get("Content-Type"); !strings.HasPrefix(mt, "application/json") {
		c.logger().ErrorContext(ctx, "响应类型错误", slog.String("content-type", mt), slog.Any("body", bytes.NewReader(bs)))
		return fmt.Errorf("响应类型错误: %s", mt)
	}

	var data SendResponse[struct{}]
	err = json.Unmarshal(bs, &data)
	if err != nil {
		c.logger().ErrorContext(ctx, "响应解析失败", slog.Any("err", err), slog.Any("body", bytes.NewReader(bs)))
		return err
	}
	if data.Code != 0 {
		c.logger().ErrorContext(ctx, "响应异常", slog.Any("code", data.Code), slog.String("msg", data.Msg))
		return fmt.Errorf("响应异常: %d %s", data.Code, data.Msg)
	}

	c.logger().InfoContext(ctx, "消息发送成功")
	return nil
}

func (c BotClient) logger() *slog.Logger { return cmp.Or(c.Logger, slog.Default()) }
func (c BotClient) client() *http.Client { return cmp.Or(c.Client, http.DefaultClient) }
func (c BotClient) baseURL() string      { return cmp.Or(c.BaseURL, "https://open.feishu.cn") }
