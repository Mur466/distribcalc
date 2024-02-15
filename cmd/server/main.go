package main

import (
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/Mur466/distribcalc/cmd/server/task"
	"github.com/gin-gonic/gin"
)

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

type Agent struct {
	AgentId    string `json:"agent_id"`
	Status     string `json:"status"`
	TotalProcs int    `json:"total_procs"`
	IdleProcs  int    `json:"idle_procs"`
	FirstSeen  time.Time
	LastSeen   time.Time
}

var router = gin.Default()
var tasks []*task.Task = make([]*task.Task, 0)
var Agents map[string]Agent = make(map[string]Agent)
var Config map[string]string = make(map[string]string)

func getAgents(c *gin.Context) {
	c.HTML(
		200,
		"agents.html",
		gin.H{
			"title":  "Agents",
			"Agents": Agents,
		},
	)
}

func pseudo_uuid() (uuid string) {

	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}

	uuid = fmt.Sprintf("%04x-%04x-%04x-%04x-%04x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])

	return
}

func getTasks(c *gin.Context) {

	c.HTML(
		200,
		"tasks.html",
		gin.H{
			"title":          "Tasks",
			"Tasks":          tasks,
			"NewRandomValue": pseudo_uuid(),
		},
	)
}

func getConfig(c *gin.Context) {
	c.HTML(
		200,
		"config.html",
		gin.H{
			"title":  "Config",
			"Config": Config,
			"TTest":  Config["test"],
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
func setConfig(c *gin.Context) {
	Config["DelayForAdd"] = ValidateDelay(c.PostForm("DelayForAdd"), Config["DelayForAdd"])
	Config["DelayForSub"] = ValidateDelay(c.PostForm("DelayForSub"), Config["DelayForSub"])
	Config["DelayForMul"] = ValidateDelay(c.PostForm("DelayForMul"), Config["DelayForMul"])
	Config["DelayForDiv"] = ValidateDelay(c.PostForm("DelayForDiv"), Config["DelayForDiv"])
	http.Redirect(c.Writer, c.Request, "/config", http.StatusSeeOther)
}

func GiveMeOperation(c *gin.Context) {
	var agent Agent
	if err := c.BindJSON(&agent); err != nil {
		fmt.Printf("Error JSON %+v", err)
		return
	}
	a, found := Agents[agent.AgentId]
	if found {
		// сохраняем старое значение
		agent.FirstSeen = a.FirstSeen
	} else {
		// инициализиуем
		agent.FirstSeen = time.Now()
	}
	agent.LastSeen = time.Now()
	Agents[agent.AgentId] = agent

	if agent.Status == "busy" {
		fmt.Printf("agent busy %+v", agent)
	} else {
		for _, t := range tasks {
			if n, ok := t.GetWaitingNodeAndSetProcess(agent.AgentId); ok {
				node := AstNode{Astnode_id: n.Astnode_id, Task_id: n.Task_id, Operand1: n.Operand1, Operand2: n.Operand2, Operator: n.Operator, Operator_delay: n.Operator_delay}
				c.IndentedJSON(http.StatusOK, node)
				fmt.Printf("agent %v received operation %+v", agent.AgentId, node)
				// дали агенту операцию, значит у него стало на 1 свободный процесс меньше
				agent.IdleProcs--
				Agents[agent.AgentId] = agent
				return
			}
		}
		// ничего нет
		c.IndentedJSON(http.StatusNoContent, AstNode{})

	}

}

func TakeOperationResult(c *gin.Context) {
	var resnode AstNode
	if err := c.BindJSON(&resnode); err != nil {
		log.Printf("Error JSON %+v", err)
		return
	}
	log.Printf("Got result %+v", resnode)
	for _, t := range tasks {
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
	tasks = append(tasks, t)
	if frombrowser{
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

func initRoutes() {

	//router.GET("/nodes", getNodes)
	//router.POST("/nodes", postNode)
	router.POST("/give-me-operation", GiveMeOperation)
	router.POST("/take-operation-result", TakeOperationResult)
	router.POST("/calculate-expression", CalculateExpression)
	router.POST("/set-config", setConfig)

	router.GET("/agents", getAgents)
	router.GET("/tasks", getTasks)
	router.GET("/config", getConfig)
	router.GET("/", getTasks)

}

func initConfig() {
	Config["DelayForAdd"] = "10"
	Config["DelayForSub"] = "12"
	Config["DelayForMul"] = "15"
	Config["DelayForDiv"] = "20"
}

func main() {
	router.LoadHTMLGlob("templates/*")
	initConfig()
	initRoutes()
	router.Run("localhost:8080")
}

/*
curl http://localhost:8080/nodes --include --header "Content-Type: application/json" --request "POST" --data "{\"Astnode_id\": 5, \"task_id\": 1, \"Operand1\": 5, \"Operand2\": 5, \"Operator\": \"*\", \"Operator_delay\" : 20}"
*/
