package router

import (
	con "backend/controller"
	"backend/middleware"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:9528", "http://127.0.0.1:9528", "http://localhost:9090", "http://127.0.0.1:9090"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Authorization", "Content-Type"},
		MaxAge:       12 * time.Hour,
	}))
	r.Static("/static", "./dist/static")
	r.StaticFile("/favicon.ico", "./dist/favicon.ico")
	r.GET("/", func(c *gin.Context) { c.File("./dist/index.html") })
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"code": 200, "message": "服务运行正常"}) })
	r.POST("/api/register", con.Register)
	r.POST("/api/login", con.Login)

	api := r.Group("/api", middleware.Auth())
	{
		api.POST("/logout", con.Logout)
		api.GET("/info", con.GetInfo)
		api.GET("/dashboard", con.Dashboard)
		api.GET("/role-options", con.RoleOptions)
		api.GET("/batches", middleware.Roles("原料供应商", "榨油厂", "运输人员", "零售商", "监管机构"), con.ListBatches)
		api.GET("/trace-batches", middleware.Roles("原料供应商", "榨油厂", "运输人员", "零售商", "监管机构"), con.ListTraceBatches)
		api.GET("/trace/:code", middleware.Roles("原料供应商", "榨油厂", "运输人员", "零售商", "监管机构"), con.TraceDetail)
		api.GET("/evidence", middleware.Roles("原料供应商", "榨油厂", "运输人员", "零售商", "监管机构"), con.ListEvidence)
		api.GET("/transport/:id/nodes", con.Nodes)

		api.POST("/batches", middleware.Roles("原料供应商"), con.CreateBatch)
		api.PUT("/batches", middleware.Roles("原料供应商"), con.UpdateBatch)
		api.DELETE("/batches", middleware.Roles("原料供应商"), con.DeleteBatch)
		api.POST("/batches/submit", middleware.Roles("原料供应商"), con.SubmitBatch)
		api.POST("/factory/decision", middleware.Roles("榨油厂"), con.FactoryDecision)
		api.POST("/factory/processing", middleware.Roles("榨油厂"), con.SubmitProcessing)
		api.POST("/transport", middleware.Roles("榨油厂"), con.CreateTransport)
		api.GET("/transport", middleware.Roles("榨油厂", "运输人员", "零售商", "监管机构"), con.ListTransports)
		api.POST("/transport/decision", middleware.Roles("运输人员"), con.TransportDecision)
		api.POST("/transport/start", middleware.Roles("运输人员"), con.StartTransport)
		api.POST("/transport/nodes", middleware.Roles("运输人员"), con.GenerateNodes)
		api.POST("/transport/complete", middleware.Roles("运输人员"), con.CompleteTransport)
		api.POST("/retail/decision", middleware.Roles("零售商"), con.RetailDecision)
		api.POST("/corrections", middleware.Roles("原料供应商", "榨油厂", "运输人员", "零售商"), con.AddCorrection)

		// Fabric 存证管理接口
		api.GET("/evidence/verify/:id", middleware.Roles("原料供应商", "榨油厂", "运输人员", "零售商", "监管机构"), con.VerifyEvidence)
		api.POST("/evidence/retry", middleware.Roles("监管机构", "系统管理员"), con.RetryEvidence)
		api.GET("/evidence/chain/:event_id", middleware.Roles("原料供应商", "榨油厂", "运输人员", "零售商", "监管机构"), con.QueryChainEvent)
		api.GET("/evidence/trace/:code", middleware.Roles("原料供应商", "榨油厂", "运输人员", "零售商", "监管机构"), con.QueryTraceChainEvents)
		api.GET("/fabric/status", middleware.Roles("监管机构", "系统管理员"), con.FabricStatus)

		api.GET("/admin/users", middleware.Roles("系统管理员"), con.ListUsers)
		api.POST("/admin/users/status", middleware.Roles("系统管理员"), con.UpdateUserStatus)
		api.GET("/admin/logs", middleware.Roles("系统管理员"), con.Logs)
	}
	r.NoRoute(func(c *gin.Context) {
		if c.Request.Method == http.MethodGet {
			c.File("./dist/index.html")
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "接口不存在"})
	})
	return r
}
