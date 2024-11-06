package abstraction

import (
	"time"
)

type Entity struct {
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}
