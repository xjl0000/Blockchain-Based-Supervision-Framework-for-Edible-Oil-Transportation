package main

import (
	"backend/blockchain"
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

	// 初始化 Fabric Gateway 连接
	if err := blockchain.Init(); err != nil {
		log.Printf("警告：Fabric Gateway 初始化失败: %v，系统将使用离线模式", err)
	} else {
		defer blockchain.GetGateway().Close()
	}

	// 启动后台重试工作器
	if blockchain.GetGateway().IsEnabled() {
		blockchain.StartRetryWorker(pkg.DB)
	}

	addr := fmt.Sprintf(":%d", viper.GetInt("app.port"))
	log.Printf("食用油运输监管系统后端已启动: http://localhost%s", addr)
	if err := router.SetupRouter().Run(addr); err != nil {
		log.Fatal(err)
	}
}
