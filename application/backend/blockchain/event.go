package blockchain

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// BusinessTypeMap 业务操作到链上事件类型的映射
var BusinessTypeMap = map[string]string{
	"原料批次存证":   "BATCH_SUBMIT",
	"原料接收存证":   "MATERIAL_ACCEPT",
	"原料拒收存证":   "MATERIAL_REJECT",
	"加工生产存证":   "PROCESSING_SUBMIT",
	"运输任务存证":   "TRANSPORT_CREATE",
	"运输接收存证":   "TRANSPORT_ACCEPT",
	"运输任务退回存证": "TRANSPORT_REJECT",
	"运输启运存证":   "TRANSPORT_START",
	"运输过程存证":   "TRANSPORT_TRACK",
	"运输完成存证":   "TRANSPORT_COMPLETE",
	"零售收货存证":   "RETAIL_ACCEPT",
	"零售退回存证":   "RETAIL_REJECT",
	"数据更正存证":   "DATA_CORRECTION",
}

// GenerateEventID 生成唯一事件ID
func GenerateEventID() string {
	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		return fmt.Sprintf("EVT-%s-%d", time.Now().Format("20060102"), time.Now().UnixNano())
	}
	return fmt.Sprintf("EVT-%s-%s", time.Now().Format("20060102"), hex.EncodeToString(random))
}

// BuildChainEvent 构建链上事件
// businessType: 中文业务类型（如"原料批次存证"）
// summary: 业务摘要
// payloadHash: 业务数据哈希
// operatorID: 操作人 ID
// role: 操作人角色
// traceCode: 溯源码
// offchainRef: 链下数据引用（如 "batches:15"）
func BuildChainEvent(businessType, summary, payloadHash string, operatorID int64, role, traceCode, offchainRef string) *ChainEvent {
	chainType, ok := BusinessTypeMap[businessType]
	if !ok {
		chainType = "UNKNOWN"
	}

	// 根据角色决定组织
	org := "Org1MSP"
	if role == "监管机构" {
		org = "Org2MSP"
	}

	return &ChainEvent{
		EventID:         GenerateEventID(),
		TraceCode:       traceCode,
		BusinessType:    chainType,
		BusinessSummary: summary,
		PayloadHash:     payloadHash,
		OperatorIDHash:  ComputeOperatorHash(operatorID),
		OperatorRole:    role,
		Organization:    org,
		EventTime:       time.Now().Format(time.RFC3339),
		OffchainRef:     offchainRef,
		SchemaVersion:   "1.0",
	}
}
