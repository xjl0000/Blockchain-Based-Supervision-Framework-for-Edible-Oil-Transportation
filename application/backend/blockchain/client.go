package blockchain

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/hyperledger/fabric-gateway/pkg/client"
)

// ChainEvent 链上事件结构（与 chaincode 中保持一致）
type ChainEvent struct {
	EventID         string `json:"event_id"`
	TraceCode       string `json:"trace_code"`
	BusinessType    string `json:"business_type"`
	BusinessSummary string `json:"business_summary"`
	PayloadHash     string `json:"payload_hash"`
	OperatorIDHash  string `json:"operator_id_hash"`
	OperatorRole    string `json:"operator_role"`
	Organization    string `json:"organization"`
	EventTime       string `json:"event_time"`
	OffchainRef     string `json:"offchain_ref"`
	SchemaVersion   string `json:"schema_version"`
	TxID            string `json:"tx_id"`
	TxTime          string `json:"tx_time"`
	SubmitterMSP    string `json:"submitter_msp"`
}

// VerifyResult 哈希核验结果
type VerifyResult struct {
	Valid     bool   `json:"valid"`
	ChainHash string `json:"chain_hash"`
	InputHash string `json:"input_hash"`
	EventID   string `json:"event_id"`
}

// SubmitResult 提交交易的返回结果
type SubmitResult struct {
	TxID        string `json:"tx_id"`
	BlockNumber uint64 `json:"block_number"`
}

// SubmitEvent 向 Fabric 提交一条业务存证
// 返回交易 ID 和区块编号
func SubmitEvent(event *ChainEvent, role string) (*SubmitResult, error) {
	gw := GetGateway()
	if !gw.IsEnabled() {
		return nil, fmt.Errorf("fabric gateway not available")
	}

	contract := gw.GetContract(role)
	if contract == nil {
		return nil, fmt.Errorf("no contract available for role: %s", role)
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("marshal event: %w", err)
	}

	log.Printf("[Fabric] 提交交易: eventID=%s, traceCode=%s, type=%s",
		event.EventID, event.TraceCode, event.BusinessType)

	// 使用 SubmitAsync 提交交易
	_, commit, err := contract.SubmitAsync("RecordEvent",
		client.WithArguments(
			event.EventID,
			event.TraceCode,
			event.BusinessType,
			event.PayloadHash,
			string(eventJSON),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("submit transaction: %w", err)
	}

	// 等待交易确认
	status, err := commit.Status()
	if err != nil {
		return nil, fmt.Errorf("get commit status: %w", err)
	}
	if !status.Successful {
		return nil, fmt.Errorf("transaction %s failed with status: %d",
			status.TransactionID, int32(status.Code))
	}

	log.Printf("[Fabric] 交易已确认: txID=%s, block=%d",
		status.TransactionID, status.BlockNumber)

	return &SubmitResult{
		TxID:        status.TransactionID,
		BlockNumber: status.BlockNumber,
	}, nil
}

// QueryEvent 从 Fabric 查询单条存证
func QueryEvent(eventID string) (*ChainEvent, error) {
	gw := GetGateway()
	if !gw.IsEnabled() {
		return nil, fmt.Errorf("fabric gateway not available")
	}

	contract := gw.GetContract("")
	if contract == nil {
		return nil, fmt.Errorf("no contract available")
	}

	result, err := contract.EvaluateTransaction("GetEvent", eventID)
	if err != nil {
		return nil, fmt.Errorf("evaluate GetEvent: %w", err)
	}

	var event ChainEvent
	if err := json.Unmarshal(result, &event); err != nil {
		return nil, fmt.Errorf("unmarshal event: %w", err)
	}

	return &event, nil
}

// QueryTraceEvents 从 Fabric 查询溯源码下全部存证
func QueryTraceEvents(traceCode string) ([]*ChainEvent, error) {
	gw := GetGateway()
	if !gw.IsEnabled() {
		return nil, fmt.Errorf("fabric gateway not available")
	}

	contract := gw.GetContract("")
	if contract == nil {
		return nil, fmt.Errorf("no contract available")
	}

	result, err := contract.EvaluateTransaction("GetTraceEvents", traceCode)
	if err != nil {
		return nil, fmt.Errorf("evaluate GetTraceEvents: %w", err)
	}

	var events []*ChainEvent
	if err := json.Unmarshal(result, &events); err != nil {
		return nil, fmt.Errorf("unmarshal events: %w", err)
	}

	return events, nil
}

// VerifyEventOnChain 在 Fabric 上核验哈希
func VerifyEventOnChain(eventID, payloadHash string) (*VerifyResult, error) {
	gw := GetGateway()
	if !gw.IsEnabled() {
		return nil, fmt.Errorf("fabric gateway not available")
	}

	contract := gw.GetContract("")
	if contract == nil {
		return nil, fmt.Errorf("no contract available")
	}

	result, err := contract.EvaluateTransaction("VerifyEvent", eventID, payloadHash)
	if err != nil {
		return nil, fmt.Errorf("evaluate VerifyEvent: %w", err)
	}

	var verifyResult VerifyResult
	if err := json.Unmarshal(result, &verifyResult); err != nil {
		return nil, fmt.Errorf("unmarshal verify result: %w", err)
	}

	return &verifyResult, nil
}
