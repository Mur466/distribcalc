package routers

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/Mur466/distribcalc/internal/handlers"
	"github.com/Mur466/distribcalc/internal/logger"
)
func InitRouters() *gin.Engine {
	router := gin.Default()
	router.Use(LoggerMiddleware(logger.Logger))
	router.LoadHTMLGlob("templates/*")


	router.POST("/give-me-operation", handlers.GiveMeOperation)
	router.POST("/take-operation-result", handlers.TakeOperationResult)
	router.POST("/calculate-expression", handlers.CalculateExpression)
	router.POST("/set-config", handlers.SetConfig)

	router.GET("/agents", handlers.GetAgents)
	router.GET("/tasks", handlers.GetTasks)
	router.GET("/config", handlers.GetConfig)
	router.GET("/", handlers.GetTasks)

	return router

}

func LoggerMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
	  // Log the request
	  logger.Info("Incoming request",
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
	  )
	  c.Next()
	}
  }