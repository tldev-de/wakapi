package models

import (
	"fmt"
	"github.com/emvi/logbuch"
	"github.com/mitchellh/hashstructure/v2"
	"strings"
	"time"
)

type Heartbeat struct {
	ID              uint64     `gorm:"primary_key" hash:"ignore"`
	User            *User      `json:"-" gorm:"not null; constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" hash:"ignore"`
	UserID          string     `json:"-" gorm:"not null; index:idx_time_user; index:idx_user_project"` // idx_user_project is for quickly fetching a user's project list (settings page)
	Entity          string     `json:"entity" gorm:"not null"`
	Type            string     `json:"type" gorm:"size:255"`
	Category        string     `json:"category" gorm:"size:255"`
	Project         string     `json:"project" gorm:"index:idx_project; index:idx_user_project"`
	Branch          string     `json:"branch" gorm:"index:idx_branch"`
	Language        string     `json:"language" gorm:"index:idx_language"`
	IsWrite         bool       `json:"is_write"`
	Editor          string     `json:"editor" gorm:"index:idx_editor" hash:"ignore"`                     // ignored because editor might be parsed differently by wakatime
	OperatingSystem string     `json:"operating_system" gorm:"index:idx_operating_system" hash:"ignore"` // ignored because os might be parsed differently by wakatime
	Machine         string     `json:"machine" gorm:"index:idx_machine" hash:"ignore"`                   // ignored because wakatime api doesn't return machines currently
	UserAgent       string     `json:"user_agent" hash:"ignore" gorm:"type:varchar(255)"`
	Time            CustomTime `json:"time" gorm:"type:timestamp(3); index:idx_time,idx_time_user" swaggertype:"primitive,number"`
	Hash            string     `json:"-" gorm:"type:varchar(17); uniqueIndex"`
	Origin          string     `json:"-" hash:"ignore" gorm:"type:varchar(255)"`
	OriginId        string     `json:"-" hash:"ignore" gorm:"type:varchar(255)"`
	CreatedAt       CustomTime `json:"created_at" gorm:"type:timestamp(3)" swaggertype:"primitive,number" hash:"ignore"` // https://gorm.io/docs/conventions.html#CreatedAt
}

func (h *Heartbeat) Valid() bool {
	return h.User != nil && h.UserID != "" && h.User.ID == h.UserID && h.Time != CustomTime(time.Time{})
}

func (h *Heartbeat) Timely(maxAge time.Duration) bool {
	now := time.Now()
	return now.Sub(h.Time.T()) <= maxAge && h.Time.T().Sub(now) < 1*time.Hour
}

func (h *Heartbeat) Augment(languageMappings map[string]string) {
	maxPrec := -1 // precision / mapping complexity -> more concrete ones shall take precedence
	for ending, value := range languageMappings {
		if ok, prec := strings.HasSuffix(h.Entity, "."+ending), strings.Count(ending, "."); ok && prec > maxPrec {
			h.Language = value
			maxPrec = prec
		}
	}
}

func (h *Heartbeat) GetKey(t uint8) (key string) {
	switch t {
	case SummaryProject:
		key = h.Project
	case SummaryEditor:
		key = h.Editor
	case SummaryLanguage:
		key = h.Language
	case SummaryOS:
		key = h.OperatingSystem
	case SummaryMachine:
		key = h.Machine
	case SummaryBranch:
		key = h.Branch
	}

	if key == "" {
		key = UnknownSummaryKey
	}

	return key
}

func (h *Heartbeat) String() string {
	return fmt.Sprintf(
		"Heartbeat {user=%s, entity=%s, type=%s, category=%s, project=%s, branch=%s, language=%s, iswrite=%v, editor=%s, os=%s, machine=%s, time=%d}",
		h.UserID,
		h.Entity,
		h.Type,
		h.Category,
		h.Project,
		h.Branch,
		h.Language,
		h.IsWrite,
		h.Editor,
		h.OperatingSystem,
		h.Machine,
		(time.Time(h.Time)).UnixNano(),
	)
}

// Hash is used to prevent duplicate heartbeats
// Using a UNIQUE INDEX over all relevant columns would be more straightforward,
// whereas manually computing this kind of hash is quite cumbersome. However,
// such a unique index would, according to https://stackoverflow.com/q/65980064/3112139,
// essentially double the space required for heartbeats, so we decided to go this way.

func (h *Heartbeat) Hashed() *Heartbeat {
	hash, err := hashstructure.Hash(h, hashstructure.FormatV2, nil)
	if err != nil {
		logbuch.Error("CRITICAL ERROR: failed to hash struct - %v", err)
	}
	h.Hash = fmt.Sprintf("%x", hash) // "uint64 values with high bit set are not supported"
	return h
}

func GetEntityColumn(t uint8) string {
	return []string{
		"project",
		"language",
		"editor",
		"operating_system",
		"machine",
		"label",
		"branch",
	}[t]
}
