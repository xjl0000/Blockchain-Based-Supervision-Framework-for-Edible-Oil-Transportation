package pkg

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// BuildEvidenceSnapshot builds the exact SQL-backed business data covered by an
// evidence hash. Data remains off-chain; only the resulting hash is submitted
// to Fabric.
func BuildEvidenceSnapshot(batchID int64, businessType, summary string, operatorID int64, operatorRole string, asOf time.Time) (map[string]interface{}, error) {
	snapshot := map[string]interface{}{
		"schema_version":   "2.0",
		"batch_id":         batchID,
		"business_type":    businessType,
		"business_summary": summary,
		"operator_id":      operatorID,
		"operator_role":    operatorRole,
	}

	batch, err := snapshotBatch(batchID)
	if err != nil {
		return nil, err
	}
	snapshot["batch"] = batch

	switch businessType {
	case "原料接收存证":
		factoryID, err := snapshotBatchParty(batchID, "oil_factory_id")
		if err != nil {
			return nil, err
		}
		snapshot["oil_factory_id"] = factoryID
	case "加工生产存证":
		factoryID, err := snapshotBatchParty(batchID, "oil_factory_id")
		if err != nil {
			return nil, err
		}
		processingData, err := snapshotBatchJSON(batchID, "processing_data")
		if err != nil {
			return nil, err
		}
		snapshot["oil_factory_id"] = factoryID
		snapshot["processing_data"] = processingData
	case "运输任务存证", "运输接收存证", "运输启运存证", "运输过程存证", "运输完成存证":
		transport, err := snapshotTransport(batchID, asOf, businessType)
		if err != nil {
			return nil, err
		}
		snapshot["transport"] = transport
	case "零售收货存证":
		retailerID, err := snapshotBatchParty(batchID, "retailer_id")
		if err != nil {
			return nil, err
		}
		receiptData, err := snapshotBatchJSON(batchID, "receipt_data")
		if err != nil {
			return nil, err
		}
		snapshot["retailer_id"] = retailerID
		snapshot["receipt_data"] = receiptData
		transport, err := snapshotTransport(batchID, asOf, businessType)
		if err != nil {
			return nil, err
		}
		snapshot["transport"] = transport
	case "原料拒收存证", "运输任务退回存证", "零售退回存证":
		if businessType == "原料拒收存证" {
			factoryID, err := snapshotBatchParty(batchID, "oil_factory_id")
			if err != nil {
				return nil, err
			}
			snapshot["oil_factory_id"] = factoryID
		}
		rejections, err := snapshotRejections(batchID, asOf)
		if err != nil {
			return nil, err
		}
		snapshot["rejections"] = rejections
	case "数据更正存证":
		corrections, err := snapshotCorrections(batchID, asOf)
		if err != nil {
			return nil, err
		}
		snapshot["corrections"] = corrections
	}

	return snapshot, nil
}

func snapshotBatch(batchID int64) (map[string]interface{}, error) {
	var id, supplierID int64
	var traceCode, materialName, origin, quantity, unit, qualityGrade, productionDate, testReport string

	err := DB.QueryRow(`SELECT id,trace_code,supplier_id,material_name,origin,CAST(quantity AS CHAR),
		unit,quality_grade,production_date,test_report
		FROM batches WHERE id=?`, batchID).Scan(
		&id, &traceCode, &supplierID, &materialName, &origin, &quantity,
		&unit, &qualityGrade, &productionDate, &testReport,
	)
	if err != nil {
		return nil, fmt.Errorf("load batch snapshot: %w", err)
	}

	return map[string]interface{}{
		"id":              id,
		"trace_code":      traceCode,
		"supplier_id":     supplierID,
		"material_name":   materialName,
		"origin":          origin,
		"quantity":        quantity,
		"unit":            unit,
		"quality_grade":   qualityGrade,
		"production_date": productionDate,
		"test_report":     testReport,
	}, nil
}

func snapshotBatchParty(batchID int64, column string) (interface{}, error) {
	if column != "oil_factory_id" && column != "transporter_id" && column != "retailer_id" {
		return nil, fmt.Errorf("unsupported batch party column: %s", column)
	}
	var value sql.NullInt64
	if err := DB.QueryRow("SELECT "+column+" FROM batches WHERE id=?", batchID).Scan(&value); err != nil {
		return nil, err
	}
	return nullIntValue(value), nil
}

func snapshotBatchJSON(batchID int64, column string) (interface{}, error) {
	if column != "processing_data" && column != "receipt_data" {
		return nil, fmt.Errorf("unsupported batch JSON column: %s", column)
	}
	var value sql.NullString
	if err := DB.QueryRow("SELECT "+column+" FROM batches WHERE id=?", batchID).Scan(&value); err != nil {
		return nil, err
	}
	return jsonValue(value), nil
}

func snapshotTransport(batchID int64, asOf time.Time, businessType string) (map[string]interface{}, error) {
	var id, factoryID, transporterID, retailerID int64
	var vehicleNo, driverName, productName, productQuantity string
	var startProvince, startCity, startLng, startLat string
	var endProvince, endCity, endLng, endLat, note string
	var createdAt time.Time
	var startedAt, completedAt sql.NullTime

	err := DB.QueryRow(`SELECT id,factory_id,transporter_id,retailer_id,vehicle_no,driver_name,
		product_name,CAST(product_quantity AS CHAR),start_province,start_city,CAST(start_lng AS CHAR),
		CAST(start_lat AS CHAR),end_province,end_city,CAST(end_lng AS CHAR),CAST(end_lat AS CHAR),
		note,created_at,started_at,completed_at FROM transport_tasks WHERE batch_id=?`, batchID).Scan(
		&id, &factoryID, &transporterID, &retailerID, &vehicleNo, &driverName,
		&productName, &productQuantity, &startProvince, &startCity, &startLng, &startLat,
		&endProvince, &endCity, &endLng, &endLat, &note, &createdAt, &startedAt, &completedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("load transport snapshot: %w", err)
	}

	transport := map[string]interface{}{
		"id":               id,
		"factory_id":       factoryID,
		"transporter_id":   transporterID,
		"retailer_id":      retailerID,
		"vehicle_no":       vehicleNo,
		"driver_name":      driverName,
		"product_name":     productName,
		"product_quantity": productQuantity,
		"start_province":   startProvince,
		"start_city":       startCity,
		"start_lng":        startLng,
		"start_lat":        startLat,
		"end_province":     endProvince,
		"end_city":         endCity,
		"end_lng":          endLng,
		"end_lat":          endLat,
		"note":             note,
		"created_at":       formatTime(createdAt),
	}

	if businessType == "运输启运存证" || businessType == "运输过程存证" || businessType == "运输完成存证" || businessType == "零售收货存证" {
		transport["started_at"] = nullTimeValue(startedAt)
	}
	if businessType == "运输完成存证" || businessType == "零售收货存证" {
		transport["completed_at"] = nullTimeValue(completedAt)
	}
	if businessType == "运输过程存证" || businessType == "运输完成存证" || businessType == "零售收货存证" {
		nodes, err := snapshotTransportNodes(id, asOf)
		if err != nil {
			return nil, err
		}
		transport["nodes"] = nodes
	}
	return transport, nil
}

func snapshotTransportNodes(taskID int64, asOf time.Time) ([]map[string]interface{}, error) {
	rows, err := DB.Query(`SELECT seq,CAST(longitude AS CHAR),CAST(latitude AS CHAR),
		CAST(temperature AS CHAR),CAST(humidity AS CHAR),recorded_at
		FROM transport_nodes WHERE task_id=? AND recorded_at<=? ORDER BY seq,id`, taskID, asOf)
	if err != nil {
		return nil, fmt.Errorf("load transport nodes snapshot: %w", err)
	}
	defer rows.Close()

	nodes := []map[string]interface{}{}
	for rows.Next() {
		var seq int
		var longitude, latitude, temperature, humidity string
		var recordedAt time.Time
		if err := rows.Scan(&seq, &longitude, &latitude, &temperature, &humidity, &recordedAt); err != nil {
			return nil, err
		}
		nodes = append(nodes, map[string]interface{}{
			"seq":         seq,
			"longitude":   longitude,
			"latitude":    latitude,
			"temperature": temperature,
			"humidity":    humidity,
			"recorded_at": formatTime(recordedAt),
		})
	}
	return nodes, rows.Err()
}

func snapshotRejections(batchID int64, asOf time.Time) ([]map[string]interface{}, error) {
	rows, err := DB.Query(`SELECT id,stage,reason,operator_id,created_at FROM rejection_records
		WHERE batch_id=? AND created_at<=? ORDER BY id`, batchID, asOf)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []map[string]interface{}{}
	for rows.Next() {
		var id, operatorID int64
		var stage, reason string
		var createdAt time.Time
		if err := rows.Scan(&id, &stage, &reason, &operatorID, &createdAt); err != nil {
			return nil, err
		}
		items = append(items, map[string]interface{}{
			"id": id, "stage": stage, "reason": reason, "operator_id": operatorID, "created_at": formatTime(createdAt),
		})
	}
	return items, rows.Err()
}

func snapshotCorrections(batchID int64, asOf time.Time) ([]map[string]interface{}, error) {
	rows, err := DB.Query(`SELECT id,stage,content,reason,operator_id,created_at FROM corrections
		WHERE batch_id=? AND created_at<=? ORDER BY id`, batchID, asOf)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []map[string]interface{}{}
	for rows.Next() {
		var id, operatorID int64
		var stage, content, reason string
		var createdAt time.Time
		if err := rows.Scan(&id, &stage, &content, &reason, &operatorID, &createdAt); err != nil {
			return nil, err
		}
		items = append(items, map[string]interface{}{
			"id": id, "stage": stage, "content": content, "reason": reason, "operator_id": operatorID, "created_at": formatTime(createdAt),
		})
	}
	return items, rows.Err()
}

func jsonValue(value sql.NullString) interface{} {
	if !value.Valid || value.String == "" || value.String == "null" {
		return nil
	}
	var decoded interface{}
	if json.Unmarshal([]byte(value.String), &decoded) != nil {
		return value.String
	}
	return decoded
}

func nullIntValue(value sql.NullInt64) interface{} {
	if value.Valid {
		return value.Int64
	}
	return nil
}

func nullTimeValue(value sql.NullTime) interface{} {
	if value.Valid {
		return formatTime(value.Time)
	}
	return nil
}

func formatTime(value time.Time) string {
	return value.Format("2006-01-02 15:04:05")
}
