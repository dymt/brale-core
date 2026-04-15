package memory

import "time"

type Outcome string

const (
	OutcomePending Outcome = "pending"
	OutcomeCorrect Outcome = "correct"
	OutcomeWrong   Outcome = "wrong"
)

type Entry struct {
	RoundID     string
	Timestamp   time.Time
	GateAction  string
	GateReason  string
	Direction   string
	Score       float64
	PriceAtTime float64
	PriceNow    float64
	ATR         float64
	KeySignal   string
	Outcome     Outcome
}

type Store interface {
	FormatForPrompt(symbol string, currentPrice float64) string
	Push(symbol string, entry Entry)
}

type Episode struct {
	ID            uint
	Symbol        string
	PositionID    string
	Direction     string
	EntryPrice    string
	ExitPrice     string
	PnLPercent    string
	Duration      string
	Reflection    string
	KeyLessons    []string
	MarketContext string
	CreatedAt     time.Time
}

type Rule struct {
	ID         uint
	Symbol     string
	RuleText   string
	Source     string
	Confidence float64
	Active     bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type EpisodicStore interface {
	ListEpisodes(symbol string, limit int) ([]Episode, error)
	SaveEpisode(episode Episode) error
	FormatForPrompt(symbol string, limit int) string
}

type SemanticStore interface {
	ListRules(symbol string, activeOnly bool, limit int) ([]Rule, error)
	SaveRule(rule Rule) error
	UpdateRule(id uint, updates map[string]any) error
	DeleteRule(id uint) error
	ToggleRule(id uint) error
	FormatForPrompt(symbol string, limit int) string
}
