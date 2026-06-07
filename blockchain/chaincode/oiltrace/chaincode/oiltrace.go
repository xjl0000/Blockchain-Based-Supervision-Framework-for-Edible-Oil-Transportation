/*
SPDX-License-Identifier: Apache-2.0

oiltrace 智能合约 — 食用油运输监管系统链码
提供四个核心接口：RecordEvent / GetEvent / GetTraceEvents / VerifyEvent
*/

package chaincode

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/hyperledger/fabric-protos-go/msp"
)

// OilTraceContract 食用油运输监管存证合约
type OilTraceContract struct {
	contractapi.Contract
}

// 常量：复合键对象类型
const traceIndexKey = "traceCode~eventID"

// RecordEvent 写入一条业务存证
// 参数：eventID, traceCode, businessType, payloadHash, eventJSON
// eventJSON 是 ChainEvent 的完整 JSON（不含 tx_id, tx_time, submitter_msp，这三个由链码自动填写）
func (s *OilTraceContract) RecordEvent(ctx contractapi.TransactionContextInterface,
	eventID string, traceCode string, businessType string, payloadHash string, eventJSON string) error {

	// 1. 检查 eventID 是否已存在
	existing, err := ctx.GetStub().GetState(eventID)
	if err != nil {
		return fmt.Errorf("failed to read state for event %s: %v", eventID, err)
	}
	if existing != nil {
		return fmt.Errorf("event %s already exists", eventID)
	}

	// 2. 解析传入的事件 JSON
	var event ChainEvent
	if err := json.Unmarshal([]byte(eventJSON), &event); err != nil {
		return fmt.Errorf("failed to parse event JSON: %v", err)
	}

	// 3. 填写链码自动生成的字段
	event.TxID = ctx.GetStub().GetTxID()

	txTimestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return fmt.Errorf("failed to get transaction timestamp: %v", err)
	}
	txTime := time.Unix(txTimestamp.Seconds, int64(txTimestamp.Nanos))
	event.TxTime = txTime.Format(time.RFC3339)

	// 获取提交者 MSP ID
	creatorBytes, err := ctx.GetStub().GetCreator()
	if err != nil {
		return fmt.Errorf("failed to get creator: %v", err)
	}
	creator := &msp.SerializedIdentity{}
	if err := proto.Unmarshal(creatorBytes, creator); err != nil {
		return fmt.Errorf("failed to parse creator identity: %v", err)
	}
	event.SubmitterMSP = creator.Mspid

	// 确保关键字段一致
	event.EventID = eventID
	event.TraceCode = traceCode
	event.BusinessType = businessType
	event.PayloadHash = payloadHash

	// 4. 序列化并保存到账本
	eventBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %v", err)
	}
	if err := ctx.GetStub().PutState(eventID, eventBytes); err != nil {
		return fmt.Errorf("failed to put state: %v", err)
	}

	// 5. 创建复合键索引（溯源码 → 事件ID），支持按溯源码查询
	indexKey, err := ctx.GetStub().CreateCompositeKey(traceIndexKey, []string{traceCode, eventID})
	if err != nil {
		return fmt.Errorf("failed to create composite key: %v", err)
	}
	// 复合键的值为空字节（只需要键本身作为索引）
	if err := ctx.GetStub().PutState(indexKey, []byte{0x00}); err != nil {
		return fmt.Errorf("failed to put composite key: %v", err)
	}

	// 6. 触发链码事件，方便外部监听
	if err := ctx.GetStub().SetEvent("RecordEvent", eventBytes); err != nil {
		return fmt.Errorf("failed to set chaincode event: %v", err)
	}

	return nil
}

// GetEvent 根据事件 ID 查询一条存证
func (s *OilTraceContract) GetEvent(ctx contractapi.TransactionContextInterface, eventID string) (*ChainEvent, error) {
	eventBytes, err := ctx.GetStub().GetState(eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to read state: %v", err)
	}
	if eventBytes == nil {
		return nil, fmt.Errorf("event %s does not exist", eventID)
	}

	var event ChainEvent
	if err := json.Unmarshal(eventBytes, &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %v", err)
	}

	return &event, nil
}

// GetTraceEvents 根据溯源码查询该批次全部链上业务事件
func (s *OilTraceContract) GetTraceEvents(ctx contractapi.TransactionContextInterface, traceCode string) ([]*ChainEvent, error) {
	// 使用复合键查询该溯源码下所有事件ID
	iterator, err := ctx.GetStub().GetStateByPartialCompositeKey(traceIndexKey, []string{traceCode})
	if err != nil {
		return nil, fmt.Errorf("failed to get state by partial composite key: %v", err)
	}
	defer iterator.Close()

	var events []*ChainEvent
	for iterator.HasNext() {
		item, err := iterator.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to iterate: %v", err)
		}

		// 从复合键中提取 eventID
		_, compositeKeyParts, err := ctx.GetStub().SplitCompositeKey(item.Key)
		if err != nil {
			return nil, fmt.Errorf("failed to split composite key: %v", err)
		}
		if len(compositeKeyParts) < 2 {
			continue
		}
		eventID := compositeKeyParts[1]

		// 根据 eventID 获取完整事件数据
		eventBytes, err := ctx.GetStub().GetState(eventID)
		if err != nil {
			return nil, fmt.Errorf("failed to get event %s: %v", eventID, err)
		}
		if eventBytes == nil {
			continue
		}

		var event ChainEvent
		if err := json.Unmarshal(eventBytes, &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal event %s: %v", eventID, err)
		}
		events = append(events, &event)
	}

	return events, nil
}

// VerifyEvent 核验传入哈希与链上哈希是否一致
func (s *OilTraceContract) VerifyEvent(ctx contractapi.TransactionContextInterface, eventID string, payloadHash string) (*VerifyResult, error) {
	eventBytes, err := ctx.GetStub().GetState(eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to read state: %v", err)
	}
	if eventBytes == nil {
		return nil, fmt.Errorf("event %s does not exist", eventID)
	}

	var event ChainEvent
	if err := json.Unmarshal(eventBytes, &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %v", err)
	}

	result := &VerifyResult{
		Valid:     event.PayloadHash == payloadHash,
		ChainHash: event.PayloadHash,
		InputHash: payloadHash,
		EventID:   eventID,
	}

	return result, nil
}
