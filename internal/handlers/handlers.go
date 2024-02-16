package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/Mur466/distribcalc/internal/task"
	"github.com/Mur466/distribcalc/internal/agent"
	"github.com/Mur466/distribcalc/internal/utils"
    l "github.com/Mur466/distribcalc/internal/logger"
)

func GetAgents(c *gin.Context) {
	c.HTML(
		200,
		"agents.html",
		gin.H{
			"title":  "Agents",
			"Agents": agent.Agents,
		},
	)
}

func GetTasks(c *gin.Context) {

	c.HTML(
		200,
		"tasks.html",
		gin.H{
			"title":          "Tasks",
			"Tasks":          task.Tasks,
			"NewRandomValue": utils.Pseudo_uuid(),
		},
	)
}

func GetConfig(c *gin.Context) {
	c.HTML(
		200,
		"config.html",
		gin.H{
			"title":  "Config",
			"Config": task.Config,
		},
	)
}
func ValidateDelay(v string, dflt string) string {
	i, err := strconv.Atoi(v)
	if v != "" && err == nil && i >= 0 {
		return v
	}
	return dflt
}
func SetConfig(c *gin.Context) {
	task.Config["DelayForAdd"] = ValidateDelay(c.PostForm("DelayForAdd"), task.Config["DelayForAdd"])
	task.Config["DelayForSub"] = ValidateDelay(c.PostForm("DelayForSub"), task.Config["DelayForSub"])
	task.Config["DelayForMul"] = ValidateDelay(c.PostForm("DelayForMul"), task.Config["DelayForMul"])
	task.Config["DelayForDiv"] = ValidateDelay(c.PostForm("DelayForDiv"), task.Config["DelayForDiv"])
	http.Redirect(c.Writer, c.Request, "/config", http.StatusSeeOther)
}

func GiveMeOperation(c *gin.Context) {
	var thisagent agent.Agent
	if err := c.BindJSON(&thisagent); err != nil {
		fmt.Printf("Error JSON %+v", err)
		return
	}
	a, found := agent.Agents[thisagent.AgentId]
	if found {
		// сохраняем старое значение
		thisagent.FirstSeen = a.FirstSeen
	} else {
		// инициализиуем
		thisagent.FirstSeen = time.Now()
	}
	thisagent.LastSeen = time.Now()
	agent.Agents[thisagent.AgentId] = thisagent

	if thisagent.Status == "busy" {
		fmt.Printf("agent busy %+v", thisagent)
	} else {
		for _, t := range task.Tasks {
			if n, ok := t.GetWaitingNodeAndSetProcess(thisagent.AgentId); ok {
				node := task.AstNode{Astnode_id: n.Astnode_id, Task_id: n.Task_id, Operand1: n.Operand1, Operand2: n.Operand2, Operator: n.Operator, Operator_delay: n.Operator_delay}
				c.IndentedJSON(http.StatusOK, node)
				fmt.Printf("agent %v received operation %+v", thisagent.AgentId, node)
				// дали агенту операцию, значит у него стало на 1 свободный процесс меньше
				thisagent.IdleProcs--
				agent.Agents[thisagent.AgentId] = thisagent
				return
			}
		}
		// ничего нет
		c.IndentedJSON(http.StatusNoContent, task.AstNode{})

	}

}

func TakeOperationResult(c *gin.Context) {
	var resnode task.AstNode
	if err := c.BindJSON(&resnode); err != nil {
		l.SLogger.Errorf("Error JSON %+v", err)
		return
	}
	l.Logger.Info("Got result",
		zap.Int("TaskId",resnode.Task_id),
		zap.Int("NodeId",resnode.Astnode_id),
		zap.String("Status",resnode.Status),
	 	zap.String("expr", fmt.Sprintf("%v%v%v=%v",resnode.Operand1, resnode.Operator, resnode.Operand2, resnode.Result)))

/*
	l.SLogger.Info("Got result",
		"TaskId",resnode.Task_id,
		"NodeId",resnode.Astnode_id,
		"Status",resnode.Status,
	 	"expr", fmt.Sprintf("%v%v%v=%v",resnode.Operand1, resnode.Operator, resnode.Operand2, resnode.Result))
*/	

	for _, t := range task.Tasks {
		if t.Task_id == resnode.Task_id {
			t.SetNodeStatus(resnode.Astnode_id, "done", task.NodeStatusInfo{Result: resnode.Result})
		}
	}

}

type ExtExpr struct {
	Ext_id string `json:"ext_id"`
	Expr   string `json:"expr"`
}

func CalculateExpression(c *gin.Context) {
	var extexpr ExtExpr
	frombrowser := false

	if c.PostForm("expr") != "" {
		// вызвали из html-формы
		extexpr.Expr = c.PostForm("expr")
		extexpr.Ext_id = c.PostForm("ext_id")
		frombrowser = true
	} else {
		// пытаемся через json
		if err := c.BindJSON(&extexpr); err == nil {
			fmt.Printf("Error JSON %+v", err)
			return
		}
	}

	t := task.NewTask(extexpr.Expr, extexpr.Ext_id)
	task.Tasks = append(task.Tasks, t)
	if frombrowser {
		http.Redirect(c.Writer, c.Request, "/tasks", http.StatusSeeOther)
	} else {
		// ответим на json
		if t.Status == "failed" {
			c.String(http.StatusBadRequest, fmt.Sprintf("Expression failed: %v", t.Message))
		} else {
			c.String(http.StatusOK, fmt.Sprintf("Expression received, task_id: %v", t.Task_id))
		}
	}

}
