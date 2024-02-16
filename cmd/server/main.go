package main

import (
	"github.com/Mur466/distribcalc/internal/task"
	"github.com/Mur466/distribcalc/internal/routers"
	"github.com/Mur466/distribcalc/internal/logger"
)



func main() {
	logger.InitLogger()
	defer logger.Logger.Sync()
	task.InitConfig()
	router := routers.InitRouters()
	router.Run("localhost:8080")
}

/*
curl http://localhost:8080/nodes --include --header "Content-Type: application/json" --request "POST" --data "{\"Astnode_id\": 5, \"task_id\": 1, \"Operand1\": 5, \"Operand2\": 5, \"Operator\": \"*\", \"Operator_delay\" : 20}"
*/
