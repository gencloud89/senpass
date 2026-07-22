package main

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"

	"nyanpass-backend/internal/config"
	"nyanpass-backend/internal/database"
	"nyanpass-backend/internal/handlers"
	"nyanpass-backend/internal/middleware"
	"nyanpass-backend/internal/models"
	"nyanpass-backend/internal/services"
)

func main() {
	cfg, err := config.Load("config.yml")
	if err != nil {
		log.Printf("[WARN] Config not found, using defaults: %v", err)
	}
	if err := database.Init(cfg.DatabasePath); err != nil {
		log.Fatalf("[FATAL] DB init: %v", err)
	}
	if err := services.SeedDefaultUser(); err != nil {
		log.Printf("[WARN] Seed user: %v", err)
	}
	var ugCount int64
	database.DB.Model(&models.UserGroup{}).Count(&ugCount)
	if ugCount == 0 {
		database.DB.Create(&models.UserGroup{Name: "普通用户", ShowOrder: 0})
		database.DB.Create(&models.UserGroup{Name: "VIP用户", ShowOrder: 1})
	}

	r := gin.Default()
	r.Use(middleware.CORS())

	// Auth
	auth := handlers.NewAuthHandler()
	r.POST("/api/v1/auth/login", auth.Login)
	r.POST("/api/v1/auth/logout", auth.Logout)

	// User info
	r.GET("/api/v1/user/info", auth.GetUserInfo)

	// Admin — cần auth + admin
	admin := r.Group("/api/v1/admin")
	admin.Use(middleware.AuthRequired(), middleware.AdminRequired())
	{
		dg := handlers.NewDeviceGroupHandler()
		folder := handlers.NewFolderHandler()
		au := handlers.NewAdminUserHandler()

		admin.GET("/devicegroup", dg.ListAll)
		admin.PUT("/devicegroup", dg.Create)
		admin.POST("/devicegroup/:id", dg.Update)
		admin.DELETE("/devicegroup", dg.Delete)
		admin.POST("/devicegroup/:id/reset_token", dg.ResetToken)
		admin.POST("/devicegroup/reset_traffic", dg.ResetTraffic)
		admin.POST("/devicegroup/reorder", dg.Reorder)
		admin.GET("/devicegroup/folder", folder.ListFolders)
		admin.PUT("/devicegroup/folder", folder.CreateFolder)
		admin.DELETE("/devicegroup/folder", folder.DeleteFolder)
		admin.POST("/devicegroup/folder/bind", folder.BindFolder)

		admin.GET("/user", au.ListUsers)
		admin.PUT("/user", au.CreateUser)
		admin.POST("/user/:id", au.UpdateUser)
		admin.DELETE("/user", au.DeleteUsers)
		admin.POST("/user/:id/reset_password", au.ResetPassword)

		admin.GET("/usergroup", au.ListUserGroups)
		admin.PUT("/usergroup", au.CreateUserGroup)
		admin.POST("/usergroup/:id", au.UpdateUserGroup)
		admin.DELETE("/usergroup", au.DeleteUserGroups)

		// Forward Rules
		fr := handlers.NewForwardRuleHandler()
		admin.GET("/forward", fr.List)
		admin.PUT("/forward", fr.Create)
		admin.POST("/forward/:id", fr.Update)
		admin.DELETE("/forward", fr.Delete)
		admin.POST("/forward/reset_traffic", fr.ResetTraffic)
		admin.POST("/forward/:id/diagnose", fr.Diagnose)
		admin.GET("/forward/folder", fr.ListFolders)
		admin.PUT("/forward/folder", fr.CreateFolder)
		admin.DELETE("/forward/folder", fr.DeleteFolder)
		admin.POST("/forward/folder/bind", fr.BindFolder)
	}

	// System
	node := handlers.NewNodeHandler()
	r.GET("/api/v1/system/node/status", node.GetNodeStatus)
	r.GET("/api/v1/system/node/status_ws", handlers.NodeStatusWS)
	r.POST("/api/v1/node/report", node.NodeReport)
	r.PUT("/api/v1/system/node/weight/:gid/:handle", node.SetWeight)
	r.POST("/api/v1/system/node/terminal/:handle", node.CreateTerminal)
	r.POST("/api/v1/system/node/kick/:handle", node.KickServer)
	// Health check
	r.GET("/api/v1/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	r.GET("/api/v1/system/info", func(c *gin.Context) {
		c.JSON(200, gin.H{"code": 0, "data": gin.H{"version": "nc20260701", "license_expire": 1784545206, "time": time.Now().Unix()}, "msg": ""})
	})

	// Client config — node client kéo cấu hình tunnel
	r.GET("/api/v1/client/config_v2", handlers.GetClientConfigV2)

	// Start WebSocket broadcast
	handlers.StartWSBroadcast()

	log.Printf("[SERVER] Starting on %s", cfg.Listen)
	if err := r.Run(cfg.Listen); err != nil {
		log.Fatalf("[FATAL] Server: %v", err)
	}
}
