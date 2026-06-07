package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

// ComputePayloadHash 对业务数据计算标准化 SHA-256 哈希
// 将数据转换为确定性 JSON（键排序），然后计算 SHA-256
func ComputePayloadHash(data map[string]interface{}) string {
	canonical := canonicalJSON(data)
	sum := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(sum[:])
}

// ComputeStringHash 对字符串计算 SHA-256
func ComputeStringHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

// ComputeOperatorHash 对操作人 ID 计算哈希（隐私保护）
func ComputeOperatorHash(operatorID int64) string {
	return ComputeStringHash(fmt.Sprintf("operator:%d", operatorID))
}

// canonicalJSON 将 map 转换为键排序后的确定性 JSON
// 确保相同数据始终生成相同的 JSON 字符串
func canonicalJSON(data interface{}) string {
	switch v := data.(type) {
	case map[string]interface{}:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		result := "{"
		for i, k := range keys {
			if i > 0 {
				result += ","
			}
			keyJSON, _ := json.Marshal(k)
			result += string(keyJSON) + ":" + canonicalJSON(v[k])
		}
		result += "}"
		return result
	case []interface{}:
		result := "["
		for i, item := range v {
			if i > 0 {
				result += ","
			}
			result += canonicalJSON(item)
		}
		result += "]"
		return result
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

// ComputeTransportNodesHash 计算运输节点整体哈希
// 将所有 GPS 节点序列化为标准 JSON 数组后计算哈希
func ComputeTransportNodesHash(nodes []map[string]interface{}) string {
	if len(nodes) == 0 {
		return ComputeStringHash("[]")
	}
	// 按 seq 排序
	sort.Slice(nodes, func(i, j int) bool {
		seqI, _ := nodes[i]["seq"].(float64)
		seqJ, _ := nodes[j]["seq"].(float64)
		return seqI < seqJ
	})
	canonical := "["
	for i, node := range nodes {
		if i > 0 {
			canonical += ","
		}
		canonical += canonicalJSON(node)
	}
	canonical += "]"
	return ComputeStringHash(canonical)
}
