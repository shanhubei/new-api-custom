package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"

	"github.com/gin-gonic/gin"
)

func SetNewuserRouter(apiRouter *gin.RouterGroup, anonymousRequestBodyLimit gin.HandlerFunc) {
	newuserRoute := apiRouter.Group("/newuser")
	newuserRoute.Use(middleware.NewuserModuleEnabled())
	{
		newuserRoute.POST("/login", middleware.CriticalRateLimit(), anonymousRequestBodyLimit, controller.NewuserLogin)
		newuserRoute.POST("/register", middleware.CriticalRateLimit(), anonymousRequestBodyLimit, controller.NewuserRegister)

		selfRoute := newuserRoute.Group("/")
		selfRoute.Use(middleware.NewuserAuth())
		{
			selfRoute.GET("/self", controller.NewuserGetSelf)
			selfRoute.GET("/token", controller.NewuserGetToken)
			selfRoute.GET("/usage", controller.NewuserGetUsage)
		}

		manageRoute := newuserRoute.Group("/manage")
		manageRoute.Use(middleware.NewuserOrgOwnerAuth())
		{
			manageRoute.GET("/users", controller.NewuserAdminList)
			manageRoute.POST("/users", middleware.CriticalRateLimit(), controller.NewuserAdminCreate)
			manageRoute.GET("/users/:id/usage", controller.NewuserAdminGetUsage)
			manageRoute.PUT("/users/:id", middleware.CriticalRateLimit(), controller.NewuserAdminUpdate)
			manageRoute.DELETE("/users/:id", controller.NewuserAdminDelete)
			manageRoute.GET("/settings", controller.NewuserAdminGetSettings)
			manageRoute.PUT("/settings", middleware.CriticalRateLimit(), controller.NewuserAdminUpdateSettings)
		}

		adminRoute := newuserRoute.Group("/admin")
		adminRoute.Use(middleware.UserAuth())
		{
			adminRoute.GET("/users", controller.NewuserAdminList)
			adminRoute.POST("/users", middleware.CriticalRateLimit(), controller.NewuserAdminCreate)
			adminRoute.GET("/users/:id/usage", controller.NewuserAdminGetUsage)
			adminRoute.PUT("/users/:id", middleware.CriticalRateLimit(), controller.NewuserAdminUpdate)
			adminRoute.DELETE("/users/:id", controller.NewuserAdminDelete)
			adminRoute.GET("/settings", controller.NewuserAdminGetSettings)
			adminRoute.PUT("/settings", middleware.CriticalRateLimit(), controller.NewuserAdminUpdateSettings)
		}
	}
}
