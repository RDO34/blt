package store

import (
	"time"

	"github.com/rdo34/blt/internal/model"
)

// Store defines persistence operations for bullets.
type Store interface {
	LoadDay(date time.Time) ([]model.Bullet, error)
	SaveDay(date time.Time, items []model.Bullet) error
	Append(date time.Time, b model.Bullet) error
	Update(date time.Time, b model.Bullet) error
	Delete(date time.Time, id string) error
}
