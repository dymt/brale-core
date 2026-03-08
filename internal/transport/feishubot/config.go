package feishubot

import "time"

const (
	ModeLongConnection = "long_connection"
	ModeCallback       = "callback"
)

type Config struct {
	AppID             string
	AppSecret         string
	Mode              string
	VerificationToken string
	EncryptKey        string
	RuntimeBaseURL    string
	SessionTTL        time.Duration
	IdempotencyTTL    time.Duration
}
