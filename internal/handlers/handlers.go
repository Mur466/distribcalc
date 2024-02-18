package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/Mur466/distribcalc/internal/agent"
	"github.com/Mur466/distribcalc/internal/cfg"
	l "github.com/Mur466/distribcalc/internal/logger"
	"github.com/Mur466/distribcalc/internal/task"
	"github.com/Mur466/distribcalc/internal/utils"
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
	// Могли бы показывать сразу task.Task, но хочется порядок новые вверху и ограничение на странице
	// поэтому берем из БД
	tasks := task.ListTasks(cfg.Cfg.RowsOnPage, 0)
	c.HTML(
		200,
		"tasks.html",
		gin.H{
			"title":          "Tasks",
			"Tasks":          tasks,
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
			"Config": cfg.Cfg,
		},
	)
}
func ValidateDelay(v string, dflt int) int {
	i, err := strconv.Atoi(v)
	if v != "" && err == nil && i >= 0 {
		return i
	}
	return dflt
}
func SetConfig(c *gin.Context) {
	cfg.Cfg.DelayForAdd = ValidateDelay(c.PostForm("DelayForAdd"), cfg.Cfg.DelayForAdd)
	cfg.Cfg.DelayForSub = ValidateDelay(c.PostForm("DelayForSub"), cfg.Cfg.DelayForSub)
	cfg.Cfg.DelayForMul = ValidateDelay(c.PostForm("DelayForMul"), cfg.Cfg.DelayForMul)
	cfg.Cfg.DelayForDiv = ValidateDelay(c.PostForm("DelayForDiv"), cfg.Cfg.DelayForDiv)
	cfg.RecalcAgentTimeout()
	cfg.Cfg.RowsOnPage = ValidateDelay(c.PostForm("RowsOnPage"), cfg.Cfg.RowsOnPage)
	http.Redirect(c.Writer, c.Request, "/config", http.StatusSeeOther)
}

func GiveMeOperation(c *gin.Context) {
	var thisagent agent.Agent
	if err := c.BindJSON(&thisagent); err != nil {
		l.SLogger.Errorf("Error JSON %+v", err)
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
		l.SLogger.Infof("Agent busy %+v", thisagent)
	} else {
		for _, t := range task.Tasks {
			if n, ok := t.GetWaitingNodeAndSetProcess(thisagent.AgentId); ok {
				node := task.Node{Node_id: n.Node_id, Task_id: n.Task_id, Operand1: n.Operand1, Operand2: n.Operand2, Operator: n.Operator, Operator_delay: n.Operator_delay}
				c.IndentedJSON(http.StatusOK, node)
				l.SLogger.Infof("agent %v received operation %+v", thisagent.AgentId, node)
				// дали агенту операцию, значит у него стало на 1 свободный процесс меньше
				thisagent.IdleProcs--
				agent.Agents[thisagent.AgentId] = thisagent
				return
			}
		}
		// ничего нет
		c.IndentedJSON(http.StatusNoContent, task.Node{})

	}

}

func TakeOperationResult(c *gin.Context) {
	var resnode task.Node
	if err := c.BindJSON(&resnode); err != nil {
		l.SLogger.Errorf("Error JSON %+v", err)
		return
	}
	l.Logger.Info("Got result",
		zap.Int("TaskId", resnode.Task_id),
		zap.Int("NodeId", resnode.Node_id),
		zap.String("Status", resnode.Status),
		zap.String("expr", fmt.Sprintf("%v%v%v=%v", resnode.Operand1, resnode.Operator, resnode.Operand2, resnode.Result)))

	for _, t := range task.Tasks {
		if t.Task_id == resnode.Task_id {
			if t.TreeSlice[resnode.Node_id].Agent_id == resnode.Agent_id {
				func() {
					t.Mx.Lock()
					defer t.Mx.Unlock()
					// тут мы 100% одни
					t.SetNodeStatus(resnode.Node_id, resnode.Status, task.NodeStatusInfo{Result: resnode.Result, Message: resnode.Message})
				}()
			} else {
				// получили результат не от того агента, который забрал операцию
				// просто проигнорируем, вдруг получим еще от кого надо
				// если не получим, то потом по таймауту повторно подадим
				l.Logger.Error("Expected result from one agent, got from another",
					zap.Int("TaskId", resnode.Task_id),
					zap.Int("NodeId", resnode.Node_id),
					zap.String("Expected agent_id", t.TreeSlice[resnode.Node_id].Agent_id),
					zap.String("Actual agent_id", resnode.Agent_id),
					zap.String("Status", resnode.Status),
					zap.String("expr", fmt.Sprintf("%v%v%v=%v", resnode.Operand1, resnode.Operator, resnode.Operand2, resnode.Result)),
				)
			}
		}
	}

}

type ExtExpr struct {
	Ext_id string `json:"ext_id"`
	Expr   string `json:"expr"`
}


type ExprResult struct {
	Ext_id  string `json:"ext_id"`
	Expr    string `json:"expr"`
	Task_id int    `json:"task_id"`
	Status  string `json:"status"`
	Result  int64  `json:"result"`
	Message string `json:"message"`
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
		if err := c.BindJSON(&extexpr); err != nil {
			l.Logger.Info("Error JSON",
				zap.String("JSON", err.Error()))
			return
		}
	}

	t := task.NewTask(extexpr.Expr, extexpr.Ext_id)
	task.Tasks[t.Task_id] = t
	if frombrowser {
		http.Redirect(c.Writer, c.Request, "/tasks", http.StatusSeeOther)
	} else {
		// ответим на json
		res := ExprResult{
			Ext_id:  t.Ext_id,
			Expr:    t.Expr,
			Task_id: t.Task_id,
			Status:  t.Status,
			Result:  t.Result,
			Message: t.Message,
		}
		c.IndentedJSON(http.StatusOK, res)
	}

}


