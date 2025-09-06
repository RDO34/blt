package model

import "time"

type BulletType string

const (
	Task                 BulletType = "task"
	Done                 BulletType = "done"
	Migrated             BulletType = "migrated"
	Scheduled            BulletType = "scheduled"
	Event                BulletType = "event"
	Note                 BulletType = "note"
	HighlightImportant   BulletType = "highlight_important"
	HighlightInspiration BulletType = "highlight_inspiration"
)

type Bullet struct {
	ID           string     `json:"id"`
	Type         BulletType `json:"type"`
	Text         string     `json:"text"`
	Tags         []string   `json:"tags,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	ScheduledFor *time.Time `json:"scheduled_for,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	Highlight    string     `json:"highlight,omitempty"`
}
