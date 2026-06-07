package controller

import (
	"backend/blockchain"
	"backend/pkg"
	"database/sql"
	"encoding/json"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

// VerifyEvidence 核验存证记录的数据一致性
// GET /api/evidence/verify/:id
func VerifyEvidence(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	// 查询该存证记录
	var eventID, dataHash, fabricTxID, fabricStatus string
	var batchID sql.NullInt64
	err := pkg.DB.QueryRow(`SELECT event_id, data_hash, fabric_tx_id, fabric_status, batch_id 
		FROM evidence_records WHERE id=?`, id).Scan(&eventID, &dataHash, &fabricTxID, &fabricStatus, &batchID)
	if err != nil {
		fail(c, 404, "存证记录不存在")
		return
	}

	result := gin.H{
		"record_id":     id,
		"event_id":      eventID,
		"fabric_status": fabricStatus,
		"fabric_tx_id":  fabricTxID,
		"mysql_hash":    dataHash,
	}

	// 如果已上链，从 Fabric 核验
	if fabricStatus == "confirmed" && eventID != "" && blockchain.GetGateway().IsEnabled() {
		// 重新计算当前 MySQL 数据的哈希
		currentHash := recomputeHash(batchID, id)

		// 在链上核验
		verifyResult, err := blockchain.VerifyEventOnChain(eventID, currentHash)
		if err != nil {
			result["verify_error"] = err.Error()
			result["verify_status"] = "无法核验"
		} else {
			result["chain_hash"] = verifyResult.ChainHash
			result["current_hash"] = currentHash
			result["verify_valid"] = verifyResult.Valid
			if verifyResult.Valid {
				result["verify_status"] = "数据一致"
			} else {
				result["verify_status"] = "数据疑似被修改"
			}
		}
	} else if fabricStatus != "confirmed" {
		result["verify_status"] = "尚未上链，无法核验"
	} else {
		result["verify_status"] = "区块链网络不可用"
	}

	ok(c, gin.H{"data": result})
}

// RetryEvidence 重新提交失败的存证记录
// POST /api/evidence/retry
func RetryEvidence(c *gin.Context) {
	var req struct {
		ID int64 `json:"id"`
	}
	_ = c.ShouldBindJSON(&req)

	// 只有监管机构可以手动重试
	if role(c) != "监管机构" && role(c) != "系统管理员" {
		fail(c, 403, "仅监管机构或管理员可以重新提交")
		return
	}

	// 查询该记录
	var fabricStatus, operatorRole string
	var payloadJSON sql.NullString
	err := pkg.DB.QueryRow(`SELECT fabric_status, operator_role, payload_json 
		FROM evidence_records WHERE id=?`, req.ID).Scan(&fabricStatus, &operatorRole, &payloadJSON)
	if err != nil {
		fail(c, 404, "存证记录不存在")
		return
	}

	if fabricStatus != "failed" && fabricStatus != "pending" {
		fail(c, 400, "仅可重试失败或待上链的记录")
		return
	}

	if !blockchain.GetGateway().IsEnabled() {
		fail(c, 503, "区块链网络当前不可用")
		return
	}

	if !payloadJSON.Valid || payloadJSON.String == "" {
		fail(c, 400, "该记录缺少链上事件数据，无法重试")
		return
	}

	// 解析事件并提交
	var event blockchain.ChainEvent
	if err := json.Unmarshal([]byte(payloadJSON.String), &event); err != nil {
		fail(c, 500, "解析事件数据失败")
		return
	}

	// 更新为提交中
	pkg.DB.Exec(`UPDATE evidence_records SET fabric_status='submitting' WHERE id=?`, req.ID)

	result, err := blockchain.SubmitEvent(&event, operatorRole)
	if err != nil {
		pkg.DB.Exec(`UPDATE evidence_records SET fabric_status='failed', 
			error_message=?, retry_count=retry_count+1 WHERE id=?`, err.Error(), req.ID)
		fail(c, 500, "上链失败："+err.Error())
		return
	}

	pkg.DB.Exec(`UPDATE evidence_records SET fabric_status='confirmed', 
		fabric_tx_id=?, fabric_block_number=?,
		fabric_channel=?, chaincode_name=?,
		transaction_hash=?, confirmed_at=NOW(), error_message=NULL WHERE id=?`,
		result.TxID, result.BlockNumber,
		viper.GetString("fabric.channel"), viper.GetString("fabric.chaincode"),
		result.TxID, req.ID)

	logAction(c, "重新上链", strconv.FormatInt(req.ID, 10))
	ok(c, gin.H{"message": "重新上链成功", "tx_id": result.TxID, "block_number": result.BlockNumber})
}

// QueryChainEvent 查询链上原始数据
// GET /api/evidence/chain/:event_id
func QueryChainEvent(c *gin.Context) {
	eventID := c.Param("event_id")

	if !blockchain.GetGateway().IsEnabled() {
		fail(c, 503, "区块链网络当前不可用")
		return
	}

	event, err := blockchain.QueryEvent(eventID)
	if err != nil {
		fail(c, 404, "链上数据查询失败："+err.Error())
		return
	}

	ok(c, gin.H{"data": event})
}

// QueryTraceChainEvents 根据溯源码查询全部链上存证
// GET /api/evidence/trace/:code
func QueryTraceChainEvents(c *gin.Context) {
	traceCode := c.Param("code")

	if !blockchain.GetGateway().IsEnabled() {
		fail(c, 503, "区块链网络当前不可用")
		return
	}

	events, err := blockchain.QueryTraceEvents(traceCode)
	if err != nil {
		fail(c, 404, "链上数据查询失败："+err.Error())
		return
	}

	ok(c, gin.H{"data": events})
}

// FabricStatus 获取 Fabric 网络状态
// GET /api/fabric/status
func FabricStatus(c *gin.Context) {
	enabled := blockchain.GetGateway().IsEnabled()

	// 统计各状态存证数量
	var pending, confirmed, failed, submitting int
	_ = pkg.DB.QueryRow("SELECT COUNT(*) FROM evidence_records WHERE fabric_status='pending'").Scan(&pending)
	_ = pkg.DB.QueryRow("SELECT COUNT(*) FROM evidence_records WHERE fabric_status='confirmed'").Scan(&confirmed)
	_ = pkg.DB.QueryRow("SELECT COUNT(*) FROM evidence_records WHERE fabric_status='failed'").Scan(&failed)
	_ = pkg.DB.QueryRow("SELECT COUNT(*) FROM evidence_records WHERE fabric_status='submitting'").Scan(&submitting)

	ok(c, gin.H{"data": gin.H{
		"fabric_enabled": enabled,
		"channel":        viper.GetString("fabric.channel"),
		"chaincode":      viper.GetString("fabric.chaincode"),
		"pending":        pending,
		"confirmed":      confirmed,
		"failed":         failed,
		"submitting":     submitting,
	}})
}

// recomputeHash 重新计算某条存证记录对应的业务数据哈希
// 用于与链上保存的哈希进行比对
func recomputeHash(batchID sql.NullInt64, recordID int64) string {
	if !batchID.Valid {
		return ""
	}

	// Rebuild the same stage-specific business snapshot from current SQL data.
	var businessType, summary, operatorRole string
	var operatorID int64
	var createdAt time.Time
	err := pkg.DB.QueryRow(`SELECT business_type, business_summary, operator_id, operator_role, created_at 
		FROM evidence_records WHERE id=?`, recordID).Scan(&businessType, &summary, &operatorID, &operatorRole, &createdAt)
	if err != nil {
		return ""
	}

	snapshot, err := pkg.BuildEvidenceSnapshot(batchID.Int64, businessType, summary, operatorID, operatorRole, createdAt)
	if err != nil {
		return ""
	}
	return blockchain.ComputePayloadHash(snapshot)
}
