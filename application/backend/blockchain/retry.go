package blockchain

import (
	"database/sql"
	"encoding/json"
	"log"
	"time"

	"github.com/spf13/viper"
)

// StartRetryWorker 启动后台重试 goroutine
// 定时扫描 fabric_status 为 'pending' 或 'failed' 的存证记录
// 重新向 Fabric 提交交易
func StartRetryWorker(db *sql.DB) {
	intervalSec := viper.GetInt("fabric.retry.interval_seconds")
	if intervalSec <= 0 {
		intervalSec = 30
	}
	maxRetries := viper.GetInt("fabric.retry.max_retries")
	if maxRetries <= 0 {
		maxRetries = 5
	}

	go func() {
		ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			if !GetGateway().IsEnabled() {
				continue
			}
			retryPendingRecords(db, maxRetries)
		}
	}()

	log.Printf("[Fabric] 重试工作器已启动: 间隔=%ds, 最大重试=%d次", intervalSec, maxRetries)
}

// retryPendingRecords 查找并重试待上链记录
func retryPendingRecords(db *sql.DB, maxRetries int) {
	rows, err := db.Query(`SELECT id, business_type, operator_role, payload_json, retry_count 
		FROM evidence_records 
		WHERE fabric_status IN ('pending', 'failed') 
		AND retry_count < ? 
		ORDER BY id ASC LIMIT 10`, maxRetries)
	if err != nil {
		log.Printf("[Fabric] 重试查询失败: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var businessType, operatorRole string
		var payloadJSON sql.NullString
		var retryCount int

		if err := rows.Scan(&id, &businessType, &operatorRole, &payloadJSON, &retryCount); err != nil {
			log.Printf("[Fabric] 重试扫描失败: %v", err)
			continue
		}

		if !payloadJSON.Valid || payloadJSON.String == "" {
			log.Printf("[Fabric] 记录 %d 缺少 payload_json，跳过", id)
			db.Exec(`UPDATE evidence_records SET fabric_status='failed', 
				error_message='payload_json 为空，无法重试', retry_count=retry_count+1 WHERE id=?`, id)
			continue
		}

		// 更新状态为提交中
		db.Exec(`UPDATE evidence_records SET fabric_status='submitting' WHERE id=?`, id)

		// 从 payload_json 中解析出事件信息并重新提交
		var event ChainEvent
		if err := json.Unmarshal([]byte(payloadJSON.String), &event); err != nil {
			log.Printf("[Fabric] 记录 %d 解析 payload 失败: %v", id, err)
			db.Exec(`UPDATE evidence_records SET fabric_status='failed', 
				error_message=?, retry_count=retry_count+1 WHERE id=?`, err.Error(), id)
			continue
		}

		result, err := SubmitEvent(&event, operatorRole)
		if err != nil {
			log.Printf("[Fabric] 记录 %d 重试失败 (第%d次): %v", id, retryCount+1, err)
			db.Exec(`UPDATE evidence_records SET fabric_status='failed', 
				error_message=?, retry_count=retry_count+1 WHERE id=?`, err.Error(), id)
			continue
		}

		// 成功
		db.Exec(`UPDATE evidence_records SET fabric_status='confirmed', 
			fabric_tx_id=?, fabric_block_number=?, 
			fabric_channel=?, chaincode_name=?,
			confirmed_at=NOW(), error_message=NULL WHERE id=?`,
			result.TxID, result.BlockNumber,
			viper.GetString("fabric.channel"), viper.GetString("fabric.chaincode"), id)
		log.Printf("[Fabric] 记录 %d 重试成功: txID=%s", id, result.TxID)
	}
}
