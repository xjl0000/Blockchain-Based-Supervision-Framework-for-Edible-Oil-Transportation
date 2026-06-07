#!/bin/bash
#
# 食用油运输监管系统 — Fabric 网络一键启动与链码部署脚本
# 使用方法：
#   cd blockchain/network
#   ./start-oiltrace.sh
#
# 前提条件：
#   1. Docker Desktop 已启动
#   2. 当前目录为 blockchain/network
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

CHANNEL_NAME="oiltracechannel"
CC_NAME="oiltrace"
CC_SRC_PATH="../chaincode/oiltrace"
CC_VERSION="1.0"

echo "============================================="
echo "  食用油运输监管系统 — Fabric v2.5 网络部署"
echo "============================================="
echo ""
echo "  Channel:   $CHANNEL_NAME"
echo "  Chaincode: $CC_NAME"
echo "  Source:     $CC_SRC_PATH"
echo ""

# Step 1: 清理旧网络（如果存在）
echo ">>> Step 1: 清理旧网络..."
./network.sh down 2>/dev/null || true
echo ""

# Step 2: 启动网络并创建通道
echo ">>> Step 2: 启动 Fabric 网络并创建通道 $CHANNEL_NAME..."
./network.sh up createChannel -c "$CHANNEL_NAME"
echo ""

# Step 3: 部署链码
echo ">>> Step 3: 部署链码 $CC_NAME..."
./network.sh deployCC -ccn "$CC_NAME" -ccp "$CC_SRC_PATH" -ccl go -c "$CHANNEL_NAME" -ccv "$CC_VERSION" -ccs 1
echo ""

# Step 4: 验证部署
echo ">>> Step 4: 验证链码部署..."
echo ""

# 设置 Org1 环境
export FABRIC_CFG_PATH=${PWD}/config/
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp
export CORE_PEER_ADDRESS=localhost:7051

# 提交测试事件
echo "  提交测试存证..."
TEST_EVENT='{"event_id":"EVT-TEST-001","trace_code":"TR-TEST-001","business_type":"TEST","business_summary":"deployment verification","payload_hash":"test123","operator_role":"admin","event_time":"2026-06-07T14:30:00+08:00","schema_version":"1.0"}'
INVOKE_ARGS=$(jq -cn --arg event "$TEST_EVENT" \
  '{function:"RecordEvent",Args:["EVT-TEST-001","TR-TEST-001","TEST","test123",$event]}')

./bin/peer chaincode invoke \
  -o localhost:7050 \
  --ordererTLSHostnameOverride orderer.example.com \
  --tls \
  --cafile "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" \
  -C "$CHANNEL_NAME" \
  -n "$CC_NAME" \
  --peerAddresses localhost:7051 \
  --tlsRootCertFiles "${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt" \
  --peerAddresses localhost:9051 \
  --tlsRootCertFiles "${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt" \
  -c "$INVOKE_ARGS" \
  --waitForEvent 2>/dev/null

echo ""
echo "  查询测试存证..."
./bin/peer chaincode query \
  -C "$CHANNEL_NAME" \
  -n "$CC_NAME" \
  -c '{"function":"GetEvent","Args":["EVT-TEST-001"]}'

echo ""
echo "============================================="
echo "  ✅ Fabric 网络部署完成！"
echo "============================================="
echo ""
echo "  网络组件："
echo "    Orderer:  localhost:7050"
echo "    Org1 Peer: localhost:7051"
echo "    Org2 Peer: localhost:9051"
echo ""
echo "  下一步：启动后端服务"
echo "    cd ../../application/backend"
echo "    go mod tidy"
echo "    go run main.go"
echo ""
