package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

type FeishuSender struct {
	client        *lark.Client
	receiveIDType string
	receiveID     string
}

func NewFeishuSender(cfg FeishuConfig) (Sender, error) {
	return newFeishuSender(cfg, nil, "")
}

func newFeishuSender(cfg FeishuConfig, httpClient *http.Client, openBaseURL string) (Sender, error) {
	if strings.TrimSpace(cfg.AppID) == "" {
		return nil, fmt.Errorf("feishu app_id is required")
	}
	if strings.TrimSpace(cfg.AppSecret) == "" {
		return nil, fmt.Errorf("feishu app_secret is required")
	}
	receiveIDType := strings.ToLower(strings.TrimSpace(cfg.DefaultReceiveIDType))
	if receiveIDType == "" {
		return nil, fmt.Errorf("feishu default_receive_id_type is required")
	}
	if strings.TrimSpace(cfg.DefaultReceiveID) == "" {
		return nil, fmt.Errorf("feishu default_receive_id is required")
	}
	if !isValidFeishuReceiveIDType(receiveIDType) {
		return nil, fmt.Errorf("feishu default_receive_id_type is invalid")
	}

	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	options := []lark.ClientOptionFunc{
		lark.WithReqTimeout(15 * time.Second),
		lark.WithHttpClient(httpClient),
	}
	if strings.TrimSpace(openBaseURL) != "" {
		options = append(options, lark.WithOpenBaseUrl(strings.TrimSpace(openBaseURL)))
	}

	client := lark.NewClient(
		cfg.AppID,
		cfg.AppSecret,
		options...,
	)

	return &FeishuSender{
		client:        client,
		receiveIDType: receiveIDType,
		receiveID:     strings.TrimSpace(cfg.DefaultReceiveID),
	}, nil
}

func (*FeishuSender) Channel() string { return "feishu" }

func (s *FeishuSender) Send(ctx context.Context, msg Message) error {
	if msg.Image != nil && len(msg.Image.Data) > 0 {
		return s.sendImage(ctx, msg)
	}
	return s.sendText(ctx, msg)
}

func (s *FeishuSender) sendText(ctx context.Context, msg Message) error {
	text := formatFeishuMessage(msg)
	if strings.TrimSpace(text) == "" {
		text = strings.TrimSpace(msg.Title)
	}
	if text == "" {
		text = "notification"
	}

	msgType := larkim.MsgTypeText
	content, err := buildFeishuTextContent(text)
	if err != nil {
		return err
	}
	if cardContent, cardErr := buildFeishuCardContent(msg, text); cardErr == nil {
		msgType = "interactive"
		content = cardContent
	}

	resp, err := s.client.Im.Message.Create(
		ctx,
		larkim.NewCreateMessageReqBuilder().
			ReceiveIdType(s.receiveIDType).
			Body(larkim.NewCreateMessageReqBodyBuilder().
				ReceiveId(s.receiveID).
				MsgType(msgType).
				Content(content).
				Build()).
			Build(),
		larkcore.WithRequestId(uuid.NewString()),
	)
	if err != nil {
		return err
	}
	if resp == nil {
		return fmt.Errorf("feishu send failed: empty response")
	}
	if !resp.Success() {
		return fmt.Errorf("feishu send failed: code=%d msg=%s request_id=%s", resp.Code, strings.TrimSpace(resp.Msg), strings.TrimSpace(resp.RequestId()))
	}
	return nil
}

func (s *FeishuSender) sendImage(ctx context.Context, msg Message) error {
	asset := msg.Image
	if asset == nil || len(asset.Data) == 0 {
		return fmt.Errorf("feishu image payload is empty")
	}
	imageKey, err := s.uploadImage(ctx, asset)
	if err != nil {
		return err
	}
	content, err := buildFeishuImageContent(imageKey)
	if err != nil {
		return err
	}
	resp, err := s.client.Im.Message.Create(
		ctx,
		larkim.NewCreateMessageReqBuilder().
			ReceiveIdType(s.receiveIDType).
			Body(larkim.NewCreateMessageReqBodyBuilder().
				ReceiveId(s.receiveID).
				MsgType("image").
				Content(content).
				Build()).
			Build(),
		larkcore.WithRequestId(uuid.NewString()),
	)
	if err != nil {
		return err
	}
	if resp == nil {
		return fmt.Errorf("feishu send image failed: empty response")
	}
	if !resp.Success() {
		return fmt.Errorf("feishu send image failed: code=%d msg=%s request_id=%s", resp.Code, strings.TrimSpace(resp.Msg), strings.TrimSpace(resp.RequestId()))
	}
	return nil
}

func (s *FeishuSender) uploadImage(ctx context.Context, asset *ImageAsset) (string, error) {
	resp, err := s.client.Im.Image.Create(
		ctx,
		larkim.NewCreateImageReqBuilder().
			Body(larkim.NewCreateImageReqBodyBuilder().
				ImageType(larkim.ImageTypeMessage).
				Image(bytes.NewReader(asset.Data)).
				Build()).
			Build(),
		larkcore.WithRequestId(uuid.NewString()),
	)
	if err != nil {
		return "", err
	}
	if !resp.Success() {
		return "", fmt.Errorf("feishu upload image failed: code=%d msg=%s request_id=%s", resp.Code, strings.TrimSpace(resp.Msg), strings.TrimSpace(resp.RequestId()))
	}
	if resp == nil || resp.Data == nil || resp.Data.ImageKey == nil || strings.TrimSpace(*resp.Data.ImageKey) == "" {
		return "", fmt.Errorf("feishu upload image failed: missing image_key")
	}
	return strings.TrimSpace(*resp.Data.ImageKey), nil
}

func formatFeishuMessage(msg Message) string {
	// 对齐 Telegram 决策报告：优先使用完整正文，而不是仅标题。
	if markdown := strings.TrimSpace(msg.Markdown); markdown != "" {
		return markdown
	}
	if plain := strings.TrimSpace(msg.Plain); plain != "" {
		return plain
	}
	if html := strings.TrimSpace(msg.HTML); html != "" {
		return html
	}
	return strings.TrimSpace(msg.Title)
}

func isValidFeishuReceiveIDType(value string) bool {
	switch value {
	case larkim.ReceiveIdTypeChatId, larkim.ReceiveIdTypeOpenId, larkim.ReceiveIdTypeUserId, larkim.ReceiveIdTypeUnionId, larkim.ReceiveIdTypeEmail:
		return true
	default:
		return false
	}
}

type feishuTextContent struct {
	Text string `json:"text"`
}

type feishuCardContent struct {
	Config   feishuCardConfig    `json:"config"`
	Header   feishuCardHeader    `json:"header"`
	Elements []feishuCardElement `json:"elements"`
}

type feishuCardHeader struct {
	Title    feishuCardTitle `json:"title"`
	Template string          `json:"template,omitempty"`
}

type feishuCardTitle struct {
	Tag     string `json:"tag"`
	Content string `json:"content"`
}

type feishuCardConfig struct {
	WideScreenMode bool `json:"wide_screen_mode"`
}

type feishuCardElement struct {
	Tag  string              `json:"tag"`
	Text *feishuCardRichText `json:"text,omitempty"`
}

type feishuCardRichText struct {
	Tag     string `json:"tag"`
	Content string `json:"content"`
}

func buildFeishuTextContent(text string) (string, error) {
	raw, err := json.Marshal(feishuTextContent{Text: text})
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func buildFeishuCardContent(msg Message, fallbackText string) (string, error) {
	title := strings.TrimSpace(msg.Title)
	if title == "" {
		title = "通知"
	}
	body := strings.TrimSpace(msg.Markdown)
	if body == "" {
		body = strings.TrimSpace(fallbackText)
	}
	if body == "" {
		body = "通知"
	}

	card := feishuCardContent{
		Config: feishuCardConfig{WideScreenMode: true},
		Header: feishuCardHeader{
			Title: feishuCardTitle{
				Tag:     "plain_text",
				Content: title,
			},
			Template: "blue",
		},
		Elements: []feishuCardElement{
			{
				Tag: "div",
				Text: &feishuCardRichText{
					Tag:     "lark_md",
					Content: body,
				},
			},
		},
	}

	raw, err := json.Marshal(card)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

type feishuImageContent struct {
	ImageKey string `json:"image_key"`
}

func buildFeishuImageContent(imageKey string) (string, error) {
	raw, err := json.Marshal(feishuImageContent{ImageKey: imageKey})
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
