package main

import (
	"backend/pkg"
	"backend/router"
	setting "backend/settings"
	"fmt"
	"log"

	"github.com/spf13/viper"
)

func main() {
	if err := setting.Init(); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	if err := pkg.InitDB(); err != nil {
		log.Fatalf("初始化 MySQL 失败: %v", err)
	}
	addr := fmt.Sprintf(":%d", viper.GetInt("app.port"))
	log.Printf("食用油运输监管系统后端已启动: http://localhost%s", addr)
	if err := router.SetupRouter().Run(addr); err != nil {
		log.Fatal(err)
	}
}
