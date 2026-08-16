package tasks

import (
	"time"
)

// Enumerado de ações
type ActionType string

const (
	ActionCommand ActionType = "command"
	ActionFile    ActionType = "file"
	ActionGo      ActionType = "go"
)

// Enumerado de tipos de gatilho
type TriggerType string

const (
	TriggerInterval TriggerType = "interval"
	TriggerFixed    TriggerType = "fixed_time"
)

// Trigger decide QUANDO a task dispara. Só um dos dois campos de valor
// é preenchido, dependendo de Tipo — isso fica mais robusto ainda no
type Trigger struct {
	// Tipo do gatilho
	// OBS.: Hórario ou Cron
	Type TriggerType `json:"type"`
	// Salva um hórario
	Interval time.Duration `json:"interval,omitempty"`
	// Data tipo cron "0 12 * * *"
	CronExpr string `json:"cron_expr,omitempty"`
}

// Estrutura de uma tarefa
type Task struct {
	Name        string     `json:"name"`
	ActionType  ActionType `json:"action_type"`
	Payload     string     `json:"payload"`
	Destination string     `json:"destination,omitempty"`
	Trigger     Trigger    `json:"trigger"`
	NextRunning time.Time  `json:"next_running"`
	LastRunning *time.Time `json:"last_running,omitempty"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
}
