/*
SPDX-License-Identifier: Apache-2.0

oiltrace chaincode — 食用油运输监管系统链码入口
*/

package main

import (
	"log"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"oiltrace/chaincode"
)

func main() {
	oiltraceChaincode, err := contractapi.NewChaincode(&chaincode.OilTraceContract{})
	if err != nil {
		log.Panicf("Error creating oiltrace chaincode: %v", err)
	}

	if err := oiltraceChaincode.Start(); err != nil {
		log.Panicf("Error starting oiltrace chaincode: %v", err)
	}
}
