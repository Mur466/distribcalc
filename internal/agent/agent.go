package agent

import (
	"time"

	"github.com/Mur466/distribcalc/internal/cfg"
	l "github.com/Mur466/distribcalc/internal/logger"
	"github.com/Mur466/distribcalc/internal/task"
	"go.uber.org/zap"
)

type Agent struct {
	AgentId    string `json:"agent_id"`
	Status     string `json:"status"`
	TotalProcs int    `json:"total_procs"`
	IdleProcs  int    `json:"idle_procs"`
	FirstSeen  time.Time
	LastSeen   time.Time
}

var Agents map[string]Agent = make(map[string]Agent)

// Удаляем пропавших агентов
func CleanLostAgents() {
	timeout := time.Second * time.Duration(cfg.Cfg.AgentLostTimeout)
	for _, a := range Agents {
		if time.Since(a.LastSeen) > timeout {
			// давно не видели, забудем про него
			l.Logger.Info("Agent lost",
				zap.String("agent_id", a.AgentId),
				zap.Time("Last seen", a.LastSeen),
				zap.Int("timeout sec", cfg.Cfg.AgentLostTimeout),
			)
			// но вначале передадим его задание другим
			for _, t := range task.Tasks {
				if t.Status == "in progress" {
					for _, n := range t.TreeSlice {
						if n.Status == "in progress" &&
							n.Agent_id == a.AgentId {
							t.SetNodeStatus(n.Node_id, "ready", task.NodeStatusInfo{})
						}
					}
				}
			}
			// нет больше такого агента
			delete(Agents, a.AgentId)
		}
	}
}

func InitAgents() {
	tick := time.NewTicker(time.Second * time.Duration(cfg.Cfg.AgentLostTimeout))
	go func() {
		for range tick.C {
			// таймер прозвенел
			CleanLostAgents()
		}

	}()

}

func (a *Agent) FirstSeenFmt() string {
	if a.FirstSeen.IsZero() {
		return "N/A"
	}
	return a.FirstSeen.Format("2006-01-02 15:04:05")
}

func (a *Agent) LastSeenFmt() string {
	if a.LastSeen.IsZero() {
		return "N/A"
	}
	//return a.LastSeen.Format("2006-01-02 15:04:05")
	return "haha"
}
