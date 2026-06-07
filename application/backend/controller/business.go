package controller

import (
	"backend/pkg"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"github.com/gin-gonic/gin"
)

func Dashboard(c *gin.Context) {
	items, err := visibleBatches(c, false)
	if err != nil {
		fail(c, 500, err.Error())
		return
	}
	total, pending, moving, completed := len(items), 0, 0, 0
	statusCounts := map[string]int{}
	for _, item := range items {
		status, _ := item["status"].(string)
		statusCounts[status]++
		if status == "pending_factory" || status == "pending_transport" || status == "pending_retail" {
			pending++
		}
		if status == "in_transit" {
			moving++
		}
		if status == "completed" {
			completed++
		}
	}
	where, args := batchVisibilityClause(c, "b", false)
	var evidence int
	_ = pkg.DB.QueryRow("SELECT COUNT(*) FROM evidence_records e JOIN batches b ON b.id=e.batch_id"+where, args...).Scan(&evidence)
	recent := items
	if len(recent) > 5 {
		recent = recent[:5]
	}
	ok(c, gin.H{"data": gin.H{"total": total, "pending": pending, "moving": moving, "completed": completed, "evidence": evidence, "status_counts": statusCounts, "recent_batches": recent}})
}

func ListBatches(c *gin.Context) {
	data, err := visibleBatches(c, true)
	if err != nil {
		fail(c, 500, err.Error())
		return
	}
	ok(c, gin.H{"data": data})
}

func ListTraceBatches(c *gin.Context) {
	data, err := visibleBatches(c, false)
	if err != nil {
		fail(c, 500, err.Error())
		return
	}
	ok(c, gin.H{"data": data})
}

func CreateBatch(c *gin.Context) {
	var req struct {
		MaterialName, Origin, Unit, QualityGrade, ProductionDate, TestReport string
		Quantity                                                             float64
	}
	if c.ShouldBindJSON(&req) != nil || req.MaterialName == "" {
		fail(c, 400, "请完整填写原料批次信息")
		return
	}
	code := pkg.GenerateTraceCode()
	res, err := pkg.DB.Exec(`INSERT INTO batches(trace_code,supplier_id,status,material_name,origin,quantity,unit,quality_grade,production_date,test_report)
		VALUES(?,?,'raw_draft',?,?,?,?,?,?,?)`, code, userID(c), req.MaterialName, req.Origin, req.Quantity, req.Unit, req.QualityGrade, req.ProductionDate, req.TestReport)
	if err != nil {
		fail(c, 500, err.Error())
		return
	}
	id, _ := res.LastInsertId()
	logAction(c, "创建原料批次", code)
	ok(c, gin.H{"message": "原料批次草稿已创建", "id": id, "trace_code": code})
}

func UpdateBatch(c *gin.Context) {
	var req struct {
		ID                                                                   int64
		MaterialName, Origin, Unit, QualityGrade, ProductionDate, TestReport string
		Quantity                                                             float64
	}
	_ = c.ShouldBindJSON(&req)
	res, err := pkg.DB.Exec(`UPDATE batches SET material_name=?,origin=?,quantity=?,unit=?,quality_grade=?,production_date=?,test_report=?
		WHERE id=? AND supplier_id=? AND status IN ('raw_draft','returned_supplier')`, req.MaterialName, req.Origin, req.Quantity, req.Unit, req.QualityGrade, req.ProductionDate, req.TestReport, req.ID, userID(c))
	if err != nil || affected(res) == 0 {
		fail(c, 400, "仅可修改自己的原料草稿")
		return
	}
	ok(c, gin.H{"message": "草稿已更新"})
}

func DeleteBatch(c *gin.Context) {
	var req struct{ ID int64 }
	_ = c.ShouldBindJSON(&req)
	res, err := pkg.DB.Exec("DELETE FROM batches WHERE id=? AND supplier_id=? AND status='raw_draft'", req.ID, userID(c))
	if err != nil || affected(res) == 0 {
		fail(c, 400, "仅可删除自己的未提交草稿")
		return
	}
	logAction(c, "删除原料草稿", strconv.FormatInt(req.ID, 10))
	ok(c, gin.H{"message": "原料草稿已删除"})
}

func SubmitBatch(c *gin.Context) {
	var req struct{ ID int64 }
	_ = c.ShouldBindJSON(&req)
	res, err := pkg.DB.Exec("UPDATE batches SET status='pending_factory' WHERE id=? AND supplier_id=? AND status IN ('raw_draft','returned_supplier')", req.ID, userID(c))
	if err != nil || affected(res) == 0 {
		fail(c, 400, "当前批次无法提交")
		return
	}
	if !createEvidenceOrFail(c, req.ID, "原料批次存证", "原料供应商提交原料批次信息") {
		return
	}
	logAction(c, "提交原料批次", strconv.FormatInt(req.ID, 10))
	ok(c, gin.H{"message": "原料信息已提交存证，等待榨油厂接收"})
}

func FactoryDecision(c *gin.Context) {
	var req struct {
		ID     int64
		Accept bool
		Reason string
	}
	_ = c.ShouldBindJSON(&req)
	if req.Accept {
		res, err := pkg.DB.Exec("UPDATE batches SET status='factory_received',oil_factory_id=? WHERE id=? AND status='pending_factory'", userID(c), req.ID)
		if err != nil || affected(res) == 0 {
			fail(c, 400, "当前批次无法接收")
			return
		}
		if !createEvidenceOrFail(c, req.ID, "原料接收存证", "榨油厂确认接收原料") {
			return
		}
		logAction(c, "确认原料接收", req.Reason)
		ok(c, gin.H{"message": "已确认接收原料"})
		return
	}
	res, err := pkg.DB.Exec("UPDATE batches SET status='returned_supplier',oil_factory_id=? WHERE id=? AND status='pending_factory'", userID(c), req.ID)
	if err != nil || affected(res) == 0 {
		fail(c, 400, "当前批次无法拒收")
		return
	}
	_, _ = pkg.DB.Exec("INSERT INTO rejection_records(batch_id,stage,reason,operator_id) VALUES(?,'原料接收',?,?)", req.ID, req.Reason, userID(c))
	if !createEvidenceOrFail(c, req.ID, "原料拒收存证", "榨油厂拒收原料："+req.Reason) {
		return
	}
	ok(c, gin.H{"message": "批次已退回原料供应商"})
}

func SubmitProcessing(c *gin.Context) {
	var req struct {
		ID   int64                  `json:"id"`
		Data map[string]interface{} `json:"data"`
	}
	_ = c.ShouldBindJSON(&req)
	payload, _ := json.Marshal(req.Data)
	res, err := pkg.DB.Exec("UPDATE batches SET processing_data=?,status='processed' WHERE id=? AND oil_factory_id=? AND status IN ('factory_received','processed')", payload, req.ID, userID(c))
	if err != nil || affected(res) == 0 {
		fail(c, 400, "当前批次无法提交加工信息")
		return
	}
	if !createEvidenceOrFail(c, req.ID, "加工生产存证", "榨油厂提交加工生产信息") {
		return
	}
	logAction(c, "提交加工信息", strconv.FormatInt(req.ID, 10))
	ok(c, gin.H{"message": "加工生产信息已提交存证"})
}

func CreateTransport(c *gin.Context) {
	var req struct {
		BatchID, TransporterID, RetailerID                                                       int64
		VehicleNo, DriverName, ProductName, StartProvince, StartCity, EndProvince, EndCity, Note string
		ProductQuantity, StartLng, StartLat, EndLng, EndLat                                      float64
	}
	_ = c.ShouldBindJSON(&req)
	if req.TransporterID == 0 || req.RetailerID == 0 {
		fail(c, 400, "请选择运输人员和目标零售商")
		return
	}
	tx, err := pkg.DB.Begin()
	if err != nil {
		fail(c, 500, "无法创建运输任务")
		return
	}
	res, err := tx.Exec(`INSERT INTO transport_tasks(batch_id,factory_id,transporter_id,retailer_id,vehicle_no,driver_name,product_name,product_quantity,start_province,start_city,start_lng,start_lat,end_province,end_city,end_lng,end_lat,status,note)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,'pending_accept',?)`, req.BatchID, userID(c), req.TransporterID, req.RetailerID, req.VehicleNo, req.DriverName, req.ProductName, req.ProductQuantity, req.StartProvince, req.StartCity, req.StartLng, req.StartLat, req.EndProvince, req.EndCity, req.EndLng, req.EndLat, req.Note)
	if err == nil {
		var update sql.Result
		update, err = tx.Exec("UPDATE batches SET status='pending_transport',transporter_id=?,retailer_id=? WHERE id=? AND oil_factory_id=? AND status='processed'", req.TransporterID, req.RetailerID, req.BatchID, userID(c))
		if err == nil && affected(update) == 0 {
			err = fmt.Errorf("batch state does not allow transport")
		}
	}
	if err != nil {
		tx.Rollback()
		fail(c, 400, "无法创建运输任务，请检查批次状态")
		return
	}
	_ = tx.Commit()
	taskID, _ := res.LastInsertId()
	if !createEvidenceOrFail(c, req.BatchID, "运输任务存证", "榨油厂发起运输任务") {
		return
	}
	logAction(c, "发起运输任务", strconv.FormatInt(taskID, 10))
	ok(c, gin.H{"message": "运输任务已发起", "task_id": taskID})
}

func ListTransports(c *gin.Context) {
	query := `SELECT t.id,t.batch_id,b.trace_code,t.vehicle_no,t.driver_name,t.product_name,t.product_quantity,t.start_province,t.start_city,t.start_lng,t.start_lat,t.end_province,t.end_city,t.end_lng,t.end_lat,t.status,t.note,t.created_at,
		f.display_name,d.display_name,r.display_name FROM transport_tasks t JOIN batches b ON b.id=t.batch_id JOIN users f ON f.id=t.factory_id JOIN users d ON d.id=t.transporter_id JOIN users r ON r.id=t.retailer_id`
	args := []interface{}{}
	switch role(c) {
	case "榨油厂":
		query += " WHERE t.factory_id=?"
		args = append(args, userID(c))
	case "运输人员":
		query += " WHERE t.transporter_id=?"
		args = append(args, userID(c))
	case "零售商":
		query += " WHERE t.retailer_id=?"
		args = append(args, userID(c))
	}
	query += " ORDER BY t.id DESC"
	rows, err := pkg.DB.Query(query, args...)
	if err != nil {
		fail(c, 500, err.Error())
		return
	}
	defer rows.Close()
	data := []gin.H{}
	for rows.Next() {
		var id, batchID int64
		var qty, slng, slat, elng, elat float64
		var trace, vehicle, driver, product, sp, sc, ep, ec, status, note, factory, transporter, retailer string
		var created interface{}
		_ = rows.Scan(&id, &batchID, &trace, &vehicle, &driver, &product, &qty, &sp, &sc, &slng, &slat, &ep, &ec, &elng, &elat, &status, &note, &created, &factory, &transporter, &retailer)
		data = append(data, gin.H{"id": id, "batch_id": batchID, "trace_code": trace, "vehicle_no": vehicle, "driver_name": driver, "product_name": product, "product_quantity": qty, "start_province": sp, "start_city": sc, "start_lng": slng, "start_lat": slat, "end_province": ep, "end_city": ec, "end_lng": elng, "end_lat": elat, "status": status, "note": note, "created_at": created, "factory": factory, "transporter": transporter, "retailer": retailer})
	}
	ok(c, gin.H{"data": data})
}

func TransportDecision(c *gin.Context) {
	var req struct {
		ID     int64
		Accept bool
		Reason string
	}
	_ = c.ShouldBindJSON(&req)
	var batchID int64
	if pkg.DB.QueryRow("SELECT batch_id FROM transport_tasks WHERE id=? AND transporter_id=? AND status='pending_accept'", req.ID, userID(c)).Scan(&batchID) != nil {
		fail(c, 400, "任务状态不允许操作")
		return
	}
	if req.Accept {
		_, _ = pkg.DB.Exec("UPDATE transport_tasks SET status='accepted' WHERE id=?", req.ID)
		_, _ = pkg.DB.Exec("UPDATE batches SET status='transport_accepted' WHERE id=?", batchID)
		if !createEvidenceOrFail(c, batchID, "运输接收存证", "运输人员确认接收任务") {
			return
		}
		ok(c, gin.H{"message": "已接收运输任务"})
		return
	}
	_, _ = pkg.DB.Exec("DELETE FROM transport_tasks WHERE id=?", req.ID)
	_, _ = pkg.DB.Exec("UPDATE batches SET status='processed' WHERE id=?", batchID)
	_, _ = pkg.DB.Exec("INSERT INTO rejection_records(batch_id,stage,reason,operator_id) VALUES(?,'运输任务',?,?)", batchID, req.Reason, userID(c))
	if !createEvidenceOrFail(c, batchID, "运输任务退回存证", "运输人员退回运输任务："+req.Reason) {
		return
	}
	ok(c, gin.H{"message": "运输任务已退回榨油厂"})
}

func StartTransport(c *gin.Context) {
	var req struct{ ID int64 }
	_ = c.ShouldBindJSON(&req)
	var batchID int64
	if pkg.DB.QueryRow("SELECT batch_id FROM transport_tasks WHERE id=? AND transporter_id=? AND status='accepted'", req.ID, userID(c)).Scan(&batchID) != nil {
		fail(c, 400, "任务无法开始")
		return
	}
	_, _ = pkg.DB.Exec("UPDATE transport_tasks SET status='in_transit',started_at=NOW() WHERE id=?", req.ID)
	_, _ = pkg.DB.Exec("UPDATE batches SET status='in_transit' WHERE id=?", batchID)
	if !createEvidenceOrFail(c, batchID, "运输启运存证", "运输人员开始运输") {
		return
	}
	ok(c, gin.H{"message": "运输已开始"})
}

func GenerateNodes(c *gin.Context) {
	var req struct {
		ID    int64
		Count int
	}
	_ = c.ShouldBindJSON(&req)
	if req.Count <= 0 {
		req.Count = 8
	}
	if req.Count > 30 {
		req.Count = 30
	}
	var batchID int64
	var startCity, endCity string
	var slng, slat, elng, elat float64
	if pkg.DB.QueryRow("SELECT batch_id,start_city,end_city,start_lng,start_lat,end_lng,end_lat FROM transport_tasks WHERE id=? AND transporter_id=? AND status='in_transit'", req.ID, userID(c)).Scan(&batchID, &startCity, &endCity, &slng, &slat, &elng, &elat) != nil {
		fail(c, 400, "仅运输中的任务可生成定位数据")
		return
	}
	var current int
	_ = pkg.DB.QueryRow("SELECT COUNT(*) FROM transport_nodes WHERE task_id=?", req.ID).Scan(&current)
	route := pkg.DomesticRoute(startCity, endCity, slng, slat, elng, elat)
	points := pkg.SampleRoute(route, current+req.Count)
	for i, point := range points[current:] {
		temp := 18 + math.Sin(float64(i))*1.4
		humidity := 52 + math.Cos(float64(i))*3.5
		_, _ = pkg.DB.Exec("INSERT INTO transport_nodes(task_id,seq,longitude,latitude,temperature,humidity) VALUES(?,?,?,?,?,?)", req.ID, current+i+1, point.Lng, point.Lat, temp, humidity)
	}
	if !createEvidenceOrFail(c, batchID, "运输过程存证", fmt.Sprintf("运输人员上传%d条GPS及温湿度记录", req.Count)) {
		return
	}
	ok(c, gin.H{"message": "运输定位与温湿度数据已更新"})
}

func CompleteTransport(c *gin.Context) {
	var req struct{ ID int64 }
	_ = c.ShouldBindJSON(&req)
	var batchID int64
	if pkg.DB.QueryRow("SELECT batch_id FROM transport_tasks WHERE id=? AND transporter_id=? AND status='in_transit'", req.ID, userID(c)).Scan(&batchID) != nil {
		fail(c, 400, "任务无法完成")
		return
	}
	_, _ = pkg.DB.Exec("UPDATE transport_tasks SET status='pending_retail',completed_at=NOW() WHERE id=?", req.ID)
	_, _ = pkg.DB.Exec("UPDATE batches SET status='pending_retail' WHERE id=?", batchID)
	if !createEvidenceOrFail(c, batchID, "运输完成存证", "运输任务完成，等待零售商收货") {
		return
	}
	ok(c, gin.H{"message": "运输已完成，等待零售商确认收货"})
}

func RetailDecision(c *gin.Context) {
	var req struct {
		BatchID int64
		Accept  bool
		Reason  string
		Data    map[string]interface{}
	}
	_ = c.ShouldBindJSON(&req)
	var taskID int64
	if pkg.DB.QueryRow("SELECT id FROM transport_tasks WHERE batch_id=? AND retailer_id=? AND status='pending_retail'", req.BatchID, userID(c)).Scan(&taskID) != nil {
		fail(c, 400, "当前批次无法收货")
		return
	}
	if req.Accept {
		payload, _ := json.Marshal(req.Data)
		_, _ = pkg.DB.Exec("UPDATE batches SET status='completed',receipt_data=? WHERE id=?", payload, req.BatchID)
		_, _ = pkg.DB.Exec("UPDATE transport_tasks SET status='completed' WHERE id=?", taskID)
		if !createEvidenceOrFail(c, req.BatchID, "零售收货存证", "零售商确认产品收货") {
			return
		}
		ok(c, gin.H{"message": "产品已确认收货，完整流程结束"})
		return
	}
	_, _ = pkg.DB.Exec("UPDATE transport_tasks SET status='in_transit',completed_at=NULL WHERE id=?", taskID)
	_, _ = pkg.DB.Exec("UPDATE batches SET status='in_transit' WHERE id=?", req.BatchID)
	_, _ = pkg.DB.Exec("INSERT INTO rejection_records(batch_id,stage,reason,operator_id) VALUES(?,'零售收货',?,?)", req.BatchID, req.Reason, userID(c))
	if !createEvidenceOrFail(c, req.BatchID, "零售退回存证", "零售商退回产品："+req.Reason) {
		return
	}
	ok(c, gin.H{"message": "产品已退回运输环节"})
}

func AddCorrection(c *gin.Context) {
	var req struct {
		BatchID                int64
		Stage, Content, Reason string
	}
	_ = c.ShouldBindJSON(&req)
	if !canAccessBatch(c, req.BatchID) {
		fail(c, 403, "无权更正该批次")
		return
	}
	_, err := pkg.DB.Exec("INSERT INTO corrections(batch_id,stage,content,reason,operator_id) VALUES(?,?,?,?,?)", req.BatchID, req.Stage, req.Content, req.Reason, userID(c))
	if err != nil {
		fail(c, 500, err.Error())
		return
	}
	if !createEvidenceOrFail(c, req.BatchID, "数据更正存证", req.Stage+"追加更正："+req.Reason) {
		return
	}
	ok(c, gin.H{"message": "更正记录已追加，原始存证保持不变"})
}

func TraceDetail(c *gin.Context) {
	code := c.Param("code")
	var id int64
	if pkg.DB.QueryRow("SELECT id FROM batches WHERE trace_code=?", code).Scan(&id) != nil {
		fail(c, 404, "未查询到该溯源码")
		return
	}
	if !canAccessBatch(c, id) {
		fail(c, 403, "无权查看该批次")
		return
	}
	batch := batchByID(id)
	evidence := evidenceByBatch(id)
	nodes := nodesByBatch(id)
	corrections := correctionsByBatch(id)
	rejections := rejectionsByBatch(id)
	transport := transportByBatch(id)
	ok(c, gin.H{"data": gin.H{"batch": batch, "transport": transport, "evidence": evidence, "nodes": nodes, "corrections": corrections, "rejections": rejections}})
}

func ListEvidence(c *gin.Context) {
	where, args := batchVisibilityClause(c, "b", false)
	query := `SELECT e.id,b.trace_code,e.business_type,e.business_summary,e.data_hash,e.previous_hash,e.transaction_hash,e.block_hash,
		e.operator_role,u.display_name,u.organization,e.created_at,
		e.fabric_status,e.fabric_tx_id,e.fabric_block_number,e.event_id,e.confirmed_at,e.error_message,e.retry_count
		FROM evidence_records e JOIN batches b ON b.id=e.batch_id JOIN users u ON u.id=e.operator_id` + where + ` ORDER BY b.id DESC,e.id`
	rows, err := pkg.DB.Query(query, args...)
	if err != nil {
		fail(c, 500, err.Error())
		return
	}
	defer rows.Close()
	data := []gin.H{}
	for rows.Next() {
		var id, fabricBlockNum int64
		var retryCount int
		var code, typ, summary, dataHash, prev, tx, block, r, operator, organization string
		var fabricStatus, fabricTxID, eventID string
		var created, confirmedAt, errorMsg interface{}
		_ = rows.Scan(&id, &code, &typ, &summary, &dataHash, &prev, &tx, &block, &r, &operator, &organization, &created,
			&fabricStatus, &fabricTxID, &fabricBlockNum, &eventID, &confirmedAt, &errorMsg, &retryCount)
		data = append(data, gin.H{"id": id, "trace_code": code, "business_type": typ, "business_summary": summary, "data_hash": dataHash, "previous_hash": prev, "transaction_hash": tx, "block_hash": block, "operator_role": r, "operator_name": operator, "operator_organization": organization, "created_at": created,
			"fabric_status": fabricStatus, "fabric_tx_id": fabricTxID, "fabric_block_number": fabricBlockNum, "event_id": eventID, "confirmed_at": confirmedAt, "error_message": errorMsg, "retry_count": retryCount})
	}
	ok(c, gin.H{"data": data})
}

func Nodes(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var batchID int64
	if pkg.DB.QueryRow("SELECT batch_id FROM transport_tasks WHERE id=?", id).Scan(&batchID) != nil || !canAccessBatch(c, batchID) {
		fail(c, 403, "无权查看该运输任务")
		return
	}
	ok(c, gin.H{"data": nodesByTask(id)})
}

func scanBatch(rows *sql.Rows) (gin.H, error) {
	var id, supplier int64
	var factory, driver, retailer sql.NullInt64
	var code, status, material, origin, unit, grade, date, report, sname, fname, dname, rname string
	var qty float64
	var processing, receipt sql.NullString
	var created, updated interface{}
	err := rows.Scan(&id, &code, &status, &material, &origin, &qty, &unit, &grade, &date, &report, &supplier, &factory, &driver, &retailer, &processing, &receipt, &created, &updated, &sname, &fname, &dname, &rname)
	processingJSON, receiptJSON := json.RawMessage("null"), json.RawMessage("null")
	if processing.Valid {
		processingJSON = json.RawMessage(processing.String)
	}
	if receipt.Valid {
		receiptJSON = json.RawMessage(receipt.String)
	}
	return gin.H{"id": id, "trace_code": code, "status": status, "material_name": material, "origin": origin, "quantity": qty, "unit": unit, "quality_grade": grade, "production_date": date, "test_report": report, "supplier_id": supplier, "factory_id": nullableInt(factory), "transporter_id": nullableInt(driver), "retailer_id": nullableInt(retailer), "processing_data": processingJSON, "receipt_data": receiptJSON, "created_at": created, "updated_at": updated, "supplier": sname, "factory": fname, "transporter": dname, "retailer": rname}, err
}
func affected(res sql.Result) int64 {
	if res == nil {
		return 0
	}
	n, _ := res.RowsAffected()
	return n
}

func createEvidenceOrFail(c *gin.Context, batchID int64, businessType, summary string) bool {
	if err := pkg.CreateEvidence(batchID, businessType, summary, userID(c), role(c)); err != nil {
		fail(c, 500, "业务数据已写入，但创建区块链存证失败："+err.Error())
		return false
	}
	return true
}
func batchByID(id int64) gin.H {
	rows, _ := pkg.DB.Query(`SELECT b.id,b.trace_code,b.status,b.material_name,b.origin,b.quantity,b.unit,b.quality_grade,b.production_date,b.test_report,b.supplier_id,b.oil_factory_id,b.transporter_id,b.retailer_id,b.processing_data,b.receipt_data,b.created_at,b.updated_at,s.display_name,COALESCE(f.display_name,''),COALESCE(d.display_name,''),COALESCE(r.display_name,'') FROM batches b JOIN users s ON s.id=b.supplier_id LEFT JOIN users f ON f.id=b.oil_factory_id LEFT JOIN users d ON d.id=b.transporter_id LEFT JOIN users r ON r.id=b.retailer_id WHERE b.id=?`, id)
	defer rows.Close()
	if rows.Next() {
		x, _ := scanBatch(rows)
		return x
	}
	return gin.H{}
}
func evidenceByBatch(id int64) []gin.H {
	rows, _ := pkg.DB.Query(`SELECT e.id,e.business_type,e.business_summary,e.data_hash,e.previous_hash,e.transaction_hash,e.block_hash,
		e.operator_role,u.display_name,u.organization,e.created_at,
		e.fabric_status,e.fabric_tx_id,e.fabric_block_number,e.event_id,e.confirmed_at
		FROM evidence_records e JOIN users u ON u.id=e.operator_id WHERE e.batch_id=? ORDER BY e.id`, id)
	defer rows.Close()
	a := []gin.H{}
	for rows.Next() {
		var eid, fabricBlockNum int64
		var t, s, dataHash, prev, tx, b, r, operator, organization string
		var fabricStatus, fabricTxID, eventID string
		var c, confirmedAt interface{}
		_ = rows.Scan(&eid, &t, &s, &dataHash, &prev, &tx, &b, &r, &operator, &organization, &c,
			&fabricStatus, &fabricTxID, &fabricBlockNum, &eventID, &confirmedAt)
		a = append(a, gin.H{"id": eid, "business_type": t, "business_summary": s, "data_hash": dataHash, "previous_hash": prev, "transaction_hash": tx, "block_hash": b, "operator_role": r, "operator_name": operator, "operator_organization": organization, "created_at": c,
			"fabric_status": fabricStatus, "fabric_tx_id": fabricTxID, "fabric_block_number": fabricBlockNum, "event_id": eventID, "confirmed_at": confirmedAt})
	}
	return a
}
func nodesByBatch(id int64) []gin.H {
	var task int64
	if pkg.DB.QueryRow("SELECT id FROM transport_tasks WHERE batch_id=?", id).Scan(&task) != nil {
		return []gin.H{}
	}
	return nodesByTask(task)
}
func nodesByTask(id int64) []gin.H {
	rows, _ := pkg.DB.Query("SELECT seq,longitude,latitude,temperature,humidity,recorded_at FROM transport_nodes WHERE task_id=? ORDER BY seq", id)
	defer rows.Close()
	a := []gin.H{}
	for rows.Next() {
		var seq int
		var lng, lat, temp, hum float64
		var c interface{}
		_ = rows.Scan(&seq, &lng, &lat, &temp, &hum, &c)
		a = append(a, gin.H{"seq": seq, "longitude": lng, "latitude": lat, "temperature": temp, "humidity": hum, "recorded_at": c})
	}
	return a
}
func correctionsByBatch(id int64) []gin.H {
	rows, _ := pkg.DB.Query("SELECT c.stage,c.content,c.reason,u.display_name,u.role,c.created_at FROM corrections c JOIN users u ON u.id=c.operator_id WHERE c.batch_id=? ORDER BY c.id", id)
	defer rows.Close()
	a := []gin.H{}
	for rows.Next() {
		var s, c, r, operator, operatorRole string
		var t interface{}
		_ = rows.Scan(&s, &c, &r, &operator, &operatorRole, &t)
		a = append(a, gin.H{"stage": s, "content": c, "reason": r, "operator_name": operator, "operator_role": operatorRole, "created_at": t})
	}
	return a
}
func rejectionsByBatch(id int64) []gin.H {
	rows, _ := pkg.DB.Query("SELECT r.stage,r.reason,u.display_name,u.role,r.created_at FROM rejection_records r JOIN users u ON u.id=r.operator_id WHERE r.batch_id=? ORDER BY r.id", id)
	defer rows.Close()
	a := []gin.H{}
	for rows.Next() {
		var s, r, operator, operatorRole string
		var t interface{}
		_ = rows.Scan(&s, &r, &operator, &operatorRole, &t)
		a = append(a, gin.H{"stage": s, "reason": r, "operator_name": operator, "operator_role": operatorRole, "created_at": t})
	}
	return a
}

func visibleBatches(c *gin.Context, includeFactoryPool bool) ([]gin.H, error) {
	query := `SELECT b.id,b.trace_code,b.status,b.material_name,b.origin,b.quantity,b.unit,b.quality_grade,b.production_date,b.test_report,
		b.supplier_id,b.oil_factory_id,b.transporter_id,b.retailer_id,b.processing_data,b.receipt_data,b.created_at,b.updated_at,
		s.display_name,COALESCE(f.display_name,''),COALESCE(d.display_name,''),COALESCE(r.display_name,'')
		FROM batches b JOIN users s ON s.id=b.supplier_id LEFT JOIN users f ON f.id=b.oil_factory_id
		LEFT JOIN users d ON d.id=b.transporter_id LEFT JOIN users r ON r.id=b.retailer_id`
	where, args := batchVisibilityClause(c, "b", includeFactoryPool)
	rows, err := pkg.DB.Query(query+where+" ORDER BY b.id DESC", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	data := []gin.H{}
	for rows.Next() {
		item, scanErr := scanBatch(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		data = append(data, item)
	}
	return data, rows.Err()
}

func batchVisibilityClause(c *gin.Context, alias string, includeFactoryPool bool) (string, []interface{}) {
	column := func(name string) string { return alias + "." + name }
	switch role(c) {
	case "原料供应商":
		return " WHERE " + column("supplier_id") + "=?", []interface{}{userID(c)}
	case "榨油厂":
		if includeFactoryPool {
			return " WHERE " + column("oil_factory_id") + "=? OR (" + column("oil_factory_id") + " IS NULL AND " + column("status") + "='pending_factory')", []interface{}{userID(c)}
		}
		return " WHERE " + column("oil_factory_id") + "=?", []interface{}{userID(c)}
	case "运输人员":
		return " WHERE " + column("transporter_id") + "=?", []interface{}{userID(c)}
	case "零售商":
		return " WHERE " + column("retailer_id") + "=?", []interface{}{userID(c)}
	case "监管机构":
		return "", nil
	case "系统管理员":
		return "", nil
	default:
		return " WHERE 1=0", nil
	}
}

func canAccessBatch(c *gin.Context, batchID int64) bool {
	if role(c) == "监管机构" {
		return true
	}
	where, args := batchVisibilityClause(c, "b", false)
	args = append([]interface{}{batchID}, args...)
	var count int
	err := pkg.DB.QueryRow("SELECT COUNT(*) FROM batches b WHERE b.id=? AND b.id IN (SELECT b.id FROM batches b"+where+")", args...).Scan(&count)
	return err == nil && count > 0
}

func transportByBatch(id int64) gin.H {
	var taskID, factoryID, transporterID, retailerID int64
	var vehicle, driver, product, sp, sc, ep, ec, status, note, factory, transporter, retailer string
	var quantity, slng, slat, elng, elat float64
	var created, started, completed sql.NullTime
	err := pkg.DB.QueryRow(`SELECT t.id,t.factory_id,t.transporter_id,t.retailer_id,t.vehicle_no,t.driver_name,t.product_name,t.product_quantity,
		t.start_province,t.start_city,t.start_lng,t.start_lat,t.end_province,t.end_city,t.end_lng,t.end_lat,t.status,t.note,
		t.created_at,t.started_at,t.completed_at,f.display_name,d.display_name,r.display_name
		FROM transport_tasks t JOIN users f ON f.id=t.factory_id JOIN users d ON d.id=t.transporter_id JOIN users r ON r.id=t.retailer_id
		WHERE t.batch_id=?`, id).Scan(&taskID, &factoryID, &transporterID, &retailerID, &vehicle, &driver, &product, &quantity,
		&sp, &sc, &slng, &slat, &ep, &ec, &elng, &elat, &status, &note, &created, &started, &completed, &factory, &transporter, &retailer)
	if err != nil {
		return gin.H{}
	}
	return gin.H{"id": taskID, "factory_id": factoryID, "transporter_id": transporterID, "retailer_id": retailerID, "vehicle_no": vehicle,
		"driver_name": driver, "product_name": product, "product_quantity": quantity, "start_province": sp, "start_city": sc,
		"start_lng": slng, "start_lat": slat, "end_province": ep, "end_city": ec, "end_lng": elng, "end_lat": elat, "status": status,
		"note": note, "created_at": nullableTime(created), "started_at": nullableTime(started), "completed_at": nullableTime(completed),
		"factory": factory, "transporter": transporter, "retailer": retailer}
}

func nullableTime(value sql.NullTime) interface{} {
	if value.Valid {
		return value.Time
	}
	return nil
}
