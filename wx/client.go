package wx

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
	MessageTypeText     MessageType = "text"     // 文本信息类型
	MessageTypeMarkdown MessageType = "markdown" // markdown 信息类型
)

// markdown 信息
type MarkdownMessage struct {
	Content string `json:"content"` // 是	markdown内容，最长不超过4096个字节，必须是utf8编码
}

// 信息
type Message struct {
	MsgType  MessageType      `json:"msgtype"`            // 信息类型
	Text     *TextMessage     `json:"text,omitempty"`     // 文本信息
	Markdown *MarkdownMessage `json:"markdown,omitempty"` // markdown 信息
}

// 文本信息
type TextMessage struct {
	Content             string   `json:"content"`                         // 是	文本内容，最长不超过2048个字节，必须是utf8编码
	MentionedList       []string `json:"mentioned_list,omitempty"`        // 否	user id 的列表，提醒群中的指定成员(@某个成员)，@all 表示提醒所有人，如果开发者获取不到 user id，可以使用 mentioned_mobile_list
	MentionedMobileList []string `json:"mentioned_mobile_list,omitempty"` // 否	手机号列表，提醒手机号对应的群成员(@某个成员)，@all 表示提醒所有人
}

// 发送响应
type SendResponse struct {
	ErrCode int    `json:"errcode"` // 错误码
	ErrMsg  string `json:"errmsg"`  // 错误说明
}

// 预定义错误
var (
	ErrNeedToken = errors.New("wx: need token") // 需要提供令牌
)

// 企业微信机器人客户端
type BotClient struct {
	Client  *http.Client // 底层 http client。不填则使用默认值。
	Logger  *slog.Logger // 日志 logger。不填则使用默认值。
	BaseURL string       // 接口基础地址。不填则使用默认值。
	Key     string       // 机器人令牌。
}

// 方法发送文本信息。
// 目前只支持文本信息。
func (c BotClient) SendText(ctx context.Context, msg string) error {
	c.logger().InfoContext(ctx, "发送文本消息", slog.String("msg", msg))

	return c.send(ctx, Message{
		MsgType: MessageTypeText,
		Text:    &TextMessage{Content: msg},
	})
}

// 发送 Markdown 信息。
func (c BotClient) SendMarkdown(ctx context.Context, msg string) error {
	c.logger().InfoContext(ctx, "发送 Markdown 消息", slog.String("msg", msg))

	return c.send(ctx, Message{
		MsgType:  MessageTypeMarkdown,
		Markdown: &MarkdownMessage{Content: msg},
	})
}

// 方法发送信息。
// 目前只支持文本信息。
func (c BotClient) Send(ctx context.Context, msg Message) error {
	c.logger().InfoContext(ctx, "发送消息", slog.String("msgType", msg.MsgType))
	return c.send(ctx, msg)
}

func (c BotClient) send(ctx context.Context, msg Message) error {
	if c.Key == "" {
		c.logger().ErrorContext(ctx, "需要提供令牌")
		return ErrNeedToken
	}

	u, err := url.Parse(c.baseURL())
	if err != nil {
		c.logger().ErrorContext(ctx, "URL 解析失败", slog.Any("err", err))
		return err
	}
	u = u.JoinPath("/cgi-bin/webhook/send")
	q := u.Query()
	q.Set("key", c.Key)
	u.RawQuery = q.Encode()

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
		c.logger().ErrorContext(ctx, "响应状态错误", slog.Int("status-code", resp.StatusCode), slog.Any("body", bytes.NewBuffer(bs)))
		return fmt.Errorf("响应状态错误: %d", resp.StatusCode)
	}
	if mt := resp.Header.Get("Content-Type"); !strings.HasPrefix(mt, "application/json") {
		c.logger().ErrorContext(ctx, "响应类型错误", slog.String("content-type", mt), slog.Any("body", bytes.NewBuffer(bs)))
		return fmt.Errorf("响应类型错误: %s", mt)
	}

	var data SendResponse
	err = json.Unmarshal(bs, &data)
	if err != nil {
		c.logger().ErrorContext(ctx, "响应解析失败", slog.Any("err", err), slog.Any("body", bytes.NewBuffer(bs)))
		return err
	}
	if data.ErrCode != 0 {
		c.logger().ErrorContext(ctx, "响应异常", slog.Any("code", data.ErrCode), slog.String("msg", data.ErrMsg))
		return fmt.Errorf("响应异常: %d %s", data.ErrCode, data.ErrMsg)
	}

	c.logger().InfoContext(ctx, "消息发送成功")
	return nil
}

func (c BotClient) logger() *slog.Logger { return cmp.Or(c.Logger, slog.Default()) }
func (c BotClient) client() *http.Client { return cmp.Or(c.Client, http.DefaultClient) }
func (c BotClient) baseURL() string      { return cmp.Or(c.BaseURL, "https://qyapi.weixin.qq.com") }
