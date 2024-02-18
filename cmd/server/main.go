package main

import (
	"strconv"

	"github.com/Mur466/distribcalc/internal/agent"
	"github.com/Mur466/distribcalc/internal/cfg"
	"github.com/Mur466/distribcalc/internal/db"
	"github.com/Mur466/distribcalc/internal/logger"
	"github.com/Mur466/distribcalc/internal/routers"
	"github.com/Mur466/distribcalc/internal/task"
)

func main() {

	cfg.InitConfig()
	logger.InitLogger()
	defer logger.Logger.Sync()

	db.InitDb()
	defer db.ShutdownDb()
	task.InitTasks()
	
	agent.InitAgents()

	router := routers.InitRouters()
	router.Run("localhost:" + strconv.Itoa(cfg.Cfg.HttpPort))
}

/*
curl http://localhost:8080/nodes --include --header "Content-Type: application/json" --request "POST" --data "{\"Astnode_id\": 5, \"task_id\": 1, \"Operand1\": 5, \"Operand2\": 5, \"Operator\": \"*\", \"Operator_delay\" : 20}"
*/
