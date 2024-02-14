package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type AstNode struct {
	Astnode_id        int `json:"astnode_id"`
	task_id           int
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

// ноды будем скармливать агентам
var nodes = []AstNode{
	{Astnode_id: 1, task_id: 1, Operand1: 1, Operand2: 1, Operator: "+", Operator_delay: 30},
	{Astnode_id: 2, task_id: 1, Operand1: 2, Operand2: 2, Operator: "*", Operator_delay: 15},
	{Astnode_id: 3, task_id: 1, Operand1: 3, Operand2: 3, Operator: "-", Operator_delay: 10},
	{Astnode_id: 4, task_id: 1, Operand1: 4, Operand2: 4, Operator: "/", Operator_delay: 15},
}

// получить список нод
func getNodes(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, nodes)
}

// добавить ноду
func postNode(c *gin.Context) {
	var newNode AstNode

	if err := c.BindJSON(&newNode); err != nil {
		return
	}

	nodes = append(nodes, newNode)
	c.IndentedJSON(http.StatusCreated, newNode)
}

type AgentStatus struct {
	AgentId    string `json:"agent_id"`
	Status     string `json:"status"`
	TotalProcs int    `json:"total_procs"`
	IdleProcs  int    `json:"idle_procs"`
}

var this int
func GiveMeOperation(c *gin.Context) {
	var agentStatus AgentStatus
	if err := c.BindJSON(&agentStatus); err != nil {
		fmt.Printf("Error JSON %+v", err)
		return
	}
	if agentStatus.Status == "busy" {
		fmt.Printf("agent busy %+v", agentStatus)
	} else {
		// ноды скармливаем с первой до последней и снова 
		if this == len(nodes) { this = 0}
		c.IndentedJSON(http.StatusOK, nodes[this])
		fmt.Printf("agent %v received operation %+v", agentStatus.AgentId, nodes[this])
		this++
	}

}


func TakeOperationResult(c *gin.Context) {
	var resnode AstNode
	if err := c.BindJSON(&resnode); err != nil {
		fmt.Printf("Error JSON %+v", err)
		return
	}
	fmt.Printf("Got result %+v", resnode)

}

func main() {
	router := gin.Default()
	router.GET("/nodes", getNodes)
	router.POST("/nodes", postNode)
	router.POST("/give-me-operation", GiveMeOperation)
	router.POST("/take-operation-result", TakeOperationResult)

	router.Run("localhost:8080")
}

/*
curl http://localhost:8080/nodes --include --header "Content-Type: application/json" --request "POST" --data "{\"Astnode_id\": 5, \"task_id\": 1, \"Operand1\": 5, \"Operand2\": 5, \"Operator\": \"*\", \"Operator_delay\" : 20}"
*/
