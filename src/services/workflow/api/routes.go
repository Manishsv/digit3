package api

import (
	"digit.org/workflow/api/handlers"
	"digit.org/workflow/config"
	"github.com/gin-gonic/gin"
)

// RegisterAllRoutes registers all API routes for the service.
func RegisterAllRoutes(
	cfg *config.Config,
	router *gin.Engine,
	processHandler *handlers.ProcessHandler,
	stateHandler *handlers.StateHandler,
	actionHandler *handlers.ActionHandler,
	transitionHandler *handlers.TransitionHandler,
	escalationConfigHandler *handlers.EscalationConfigHandler,
	autoEscalationHandler *handlers.AutoEscalationHandler,
) {
	v1 := router.Group(cfg.Server.ContextPath)

	// Process routes
	processGroup := v1.Group("/v1/process")
	{
		processGroup.POST("", processHandler.CreateProcess)
		processGroup.GET("", processHandler.GetProcesses)
		processGroup.DELETE("", processHandler.DeleteProcessByQuery)
		processGroup.GET("/definition", processHandler.GetProcessDefinitions) // New route
		processGroup.GET("/:id", processHandler.GetProcess)
		processGroup.PUT("/:id", processHandler.UpdateProcess)
		processGroup.DELETE("/:id", processHandler.DeleteProcess)

		// Nested State routes
		processGroup.POST("/:id/state", stateHandler.CreateState)
		processGroup.GET("/:id/state", stateHandler.GetStates)

		// Nested Escalation Config routes
		processGroup.POST("/:id/escalation", escalationConfigHandler.CreateEscalationConfig)
		processGroup.GET("/:id/escalation", escalationConfigHandler.GetEscalationConfigs)
	}

	// State routes (for operations on a state by its own ID)
	stateGroup := v1.Group("/v1/state")
	{
		stateGroup.GET("/:id", stateHandler.GetState)
		stateGroup.PUT("/:id", stateHandler.UpdateState)
		stateGroup.DELETE("/:id", stateHandler.DeleteState)

		// Nested Action routes
		stateGroup.POST("/:id/action", actionHandler.CreateAction)
		stateGroup.GET("/:id/action", actionHandler.GetActions)
	}

	// Action routes (for operations on an action by its own ID)
	actionGroup := v1.Group("/v1/action")
	{
		actionGroup.GET("/:id", actionHandler.GetAction)
		actionGroup.PUT("/:id", actionHandler.UpdateAction)
		actionGroup.DELETE("/:id", actionHandler.DeleteAction)
	}

	// Escalation Config routes (for operations on an escalation config by its own ID)
	escalationGroup := v1.Group("/v1/escalation")
	{
		escalationGroup.GET("/:id", escalationConfigHandler.GetEscalationConfig)
		escalationGroup.PUT("/:id", escalationConfigHandler.UpdateEscalationConfig)
		escalationGroup.DELETE("/:id", escalationConfigHandler.DeleteEscalationConfig)
	}

	// Transition routes
	v1.POST("/v1/transition", transitionHandler.Transition)
	v1.GET("/v1/transition", transitionHandler.GetTransitions)

	// Auto-escalation routes
	v1.POST("/v1/auto/:processCode/_escalate", autoEscalationHandler.EscalateApplications)
	v1.GET("/v1/auto/_search", autoEscalationHandler.SearchEscalatedApplications)
}
