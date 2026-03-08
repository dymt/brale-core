package notify

import (
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

func (s *FeishuSender) Send(ctx context.Context, msg Message) error {
	text := formatFeishuMessage(msg)
	if strings.TrimSpace(text) == "" {
		text = strings.TrimSpace(msg.Title)
	}
	if text == "" {
		text = "notification"
	}

	content, err := buildFeishuTextContent(text)
	if err != nil {
		return err
	}
	resp, err := s.client.Im.Message.Create(
		ctx,
		larkim.NewCreateMessageReqBuilder().
			ReceiveIdType(s.receiveIDType).
			Body(larkim.NewCreateMessageReqBodyBuilder().
				ReceiveId(s.receiveID).
				MsgType(larkim.MsgTypeText).
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

func formatFeishuMessage(msg Message) string {
	if plain := strings.TrimSpace(msg.Plain); plain != "" {
		return plain
	}
	if markdown := strings.TrimSpace(msg.Markdown); markdown != "" {
		return markdown
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

func buildFeishuTextContent(text string) (string, error) {
	raw, err := json.Marshal(feishuTextContent{Text: text})
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
