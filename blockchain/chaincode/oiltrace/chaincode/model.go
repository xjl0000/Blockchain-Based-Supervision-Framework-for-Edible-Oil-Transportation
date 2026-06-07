/*
SPDX-License-Identifier: Apache-2.0

链上事件数据模型
*/

package chaincode

// ChainEvent 存储在 Fabric 账本中的业务事件存证
type ChainEvent struct {
	EventID         string `json:"event_id"`          // 事件唯一标识，如 EVT-20260607-000001
	TraceCode       string `json:"trace_code"`        // 溯源码，如 TR202606020002
	BusinessType    string `json:"business_type"`     // 业务类型，如 BATCH_SUBMIT
	BusinessSummary string `json:"business_summary"`  // 业务摘要，如 "原料供应商提交原料批次信息"
	PayloadHash     string `json:"payload_hash"`      // 业务数据 SHA-256 哈希
	OperatorIDHash  string `json:"operator_id_hash"`  // 操作人 ID 哈希（隐私保护）
	OperatorRole    string `json:"operator_role"`     // 操作人角色
	Organization    string `json:"organization"`      // 提交组织 MSP ID
	EventTime       string `json:"event_time"`        // 业务事件发生时间（ISO 8601）
	OffchainRef     string `json:"offchain_ref"`      // 链下数据引用，如 "batches:15"
	SchemaVersion   string `json:"schema_version"`    // 数据结构版本
	TxID            string `json:"tx_id"`             // Fabric 交易 ID（链码内自动填写）
	TxTime          string `json:"tx_time"`           // Fabric 交易时间（链码内自动填写）
	SubmitterMSP    string `json:"submitter_msp"`     // 实际提交者 MSP ID（链码内自动填写）
}

// VerifyResult 哈希核验结果
type VerifyResult struct {
	Valid     bool   `json:"valid"`      // 哈希是否一致
	ChainHash string `json:"chain_hash"` // 链上保存的哈希
	InputHash string `json:"input_hash"` // 传入的哈希
	EventID   string `json:"event_id"`   // 事件 ID
}

// TraceIndex 溯源码 → 事件 ID 的索引记录
// 用于支持按溯源码查询全部事件
type TraceIndex struct {
	TraceCode string   `json:"trace_code"`
	EventIDs  []string `json:"event_ids"`
}
