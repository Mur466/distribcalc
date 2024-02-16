package agent

import "time"
/*
type AstNode struct {
	Astnode_id        int `json:"astnode_id"`
	Task_id           int `json:"task_id"`
	parent_astnode_id int
	Operand1          int    `json:"operand1"`
	Operand2          int    `json:"operand2"`
	Operator          string `json:"operator"`
	Operator_delay    int    `json:"operator_delay"`
	status            string
	date_ins          time.Time
	date_start        time.Time
	date_done         time.Time
	agent_id          int
	Result            int64 `json:"result"`
}
*/
type Agent struct {
	AgentId    string `json:"agent_id"`
	Status     string `json:"status"`
	TotalProcs int    `json:"total_procs"`
	IdleProcs  int    `json:"idle_procs"`
	FirstSeen  time.Time
	LastSeen   time.Time
}

var Agents map[string]Agent = make(map[string]Agent)
