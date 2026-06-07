package pkg

import (
	"backend/blockchain"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-jwt/jwt"
	"github.com/spf13/viper"
)

var DB *sql.DB

const TokenExpireDuration = 48 * time.Hour

type Claims struct {
	UserID   int64  `json:"userID"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.StandardClaims
}

func InitDB() error {
	serverDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=true&loc=Local",
		viper.GetString("mysql.user"), viper.GetString("mysql.password"),
		viper.GetString("mysql.host"), viper.GetString("mysql.port"))
	root, err := sql.Open("mysql", serverDSN)
	if err != nil {
		return err
	}
	defer root.Close()
	dbName := viper.GetString("mysql.db")
	if _, err = root.Exec("CREATE DATABASE IF NOT EXISTS `" + dbName + "` DEFAULT CHARACTER SET utf8mb4"); err != nil {
		return err
	}
	databaseDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=Local",
		viper.GetString("mysql.user"), viper.GetString("mysql.password"),
		viper.GetString("mysql.host"), viper.GetString("mysql.port"), dbName)
	DB, err = sql.Open("mysql", databaseDSN)
	if err != nil {
		return err
	}
	if err = DB.Ping(); err != nil {
		return err
	}
	for _, stmt := range schema {
		if _, err = DB.Exec(stmt); err != nil {
			return fmt.Errorf("create schema: %w", err)
		}
	}
	// 运行 Fabric 字段增量迁移（对已存在的表添加新字段，忽略已存在错误）
	for _, stmt := range fabricMigrations {
		DB.Exec(stmt) // 忽略错误，字段可能已存在
	}
	return seedData()
}

var schema = []string{
	`CREATE TABLE IF NOT EXISTS users (
		id BIGINT PRIMARY KEY AUTO_INCREMENT, username VARCHAR(50) UNIQUE NOT NULL,
		password VARCHAR(64) NOT NULL, display_name VARCHAR(80) NOT NULL, role VARCHAR(30) NOT NULL,
		organization VARCHAR(120) NOT NULL DEFAULT '', phone VARCHAR(30) NOT NULL DEFAULT '',
		status VARCHAR(20) NOT NULL DEFAULT 'pending', review_note VARCHAR(255) NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS batches (
		id BIGINT PRIMARY KEY AUTO_INCREMENT, trace_code VARCHAR(32) UNIQUE NOT NULL, supplier_id BIGINT NOT NULL,
		oil_factory_id BIGINT NULL, transporter_id BIGINT NULL, retailer_id BIGINT NULL,
		status VARCHAR(40) NOT NULL DEFAULT 'raw_draft', material_name VARCHAR(80) NOT NULL,
		origin VARCHAR(120) NOT NULL DEFAULT '', quantity DECIMAL(12,2) NOT NULL DEFAULT 0, unit VARCHAR(20) NOT NULL DEFAULT '吨',
		quality_grade VARCHAR(40) NOT NULL DEFAULT '', production_date VARCHAR(30) NOT NULL DEFAULT '',
		test_report VARCHAR(255) NOT NULL DEFAULT '', processing_data JSON NULL, receipt_data JSON NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS transport_tasks (
		id BIGINT PRIMARY KEY AUTO_INCREMENT, batch_id BIGINT UNIQUE NOT NULL, factory_id BIGINT NOT NULL,
		transporter_id BIGINT NOT NULL, retailer_id BIGINT NOT NULL, vehicle_no VARCHAR(30) NOT NULL,
		driver_name VARCHAR(80) NOT NULL, product_name VARCHAR(100) NOT NULL, product_quantity DECIMAL(12,2) NOT NULL DEFAULT 0,
		start_province VARCHAR(40) NOT NULL, start_city VARCHAR(40) NOT NULL, start_lng DECIMAL(10,6) NOT NULL, start_lat DECIMAL(10,6) NOT NULL,
		end_province VARCHAR(40) NOT NULL, end_city VARCHAR(40) NOT NULL, end_lng DECIMAL(10,6) NOT NULL, end_lat DECIMAL(10,6) NOT NULL,
		status VARCHAR(30) NOT NULL DEFAULT 'pending_accept', note VARCHAR(255) NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, started_at DATETIME NULL, completed_at DATETIME NULL
	)`,
	`CREATE TABLE IF NOT EXISTS transport_nodes (
		id BIGINT PRIMARY KEY AUTO_INCREMENT, task_id BIGINT NOT NULL, seq INT NOT NULL,
		longitude DECIMAL(10,6) NOT NULL, latitude DECIMAL(10,6) NOT NULL,
		temperature DECIMAL(5,2) NOT NULL, humidity DECIMAL(5,2) NOT NULL,
		recorded_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS corrections (
		id BIGINT PRIMARY KEY AUTO_INCREMENT, batch_id BIGINT NOT NULL, stage VARCHAR(30) NOT NULL,
		content TEXT NOT NULL, reason VARCHAR(255) NOT NULL, operator_id BIGINT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS rejection_records (
		id BIGINT PRIMARY KEY AUTO_INCREMENT, batch_id BIGINT NOT NULL, stage VARCHAR(30) NOT NULL,
		reason VARCHAR(255) NOT NULL, operator_id BIGINT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS evidence_records (
		id BIGINT PRIMARY KEY AUTO_INCREMENT, batch_id BIGINT NULL, business_type VARCHAR(50) NOT NULL,
		business_summary VARCHAR(255) NOT NULL, data_hash VARCHAR(64) NOT NULL, previous_hash VARCHAR(64) NOT NULL DEFAULT '',
		transaction_hash VARCHAR(64) NOT NULL DEFAULT '', block_hash VARCHAR(64) NOT NULL DEFAULT '',
		operator_id BIGINT NOT NULL, operator_role VARCHAR(30) NOT NULL,
		fabric_status VARCHAR(20) NOT NULL DEFAULT 'pending',
		fabric_tx_id VARCHAR(128) NOT NULL DEFAULT '',
		fabric_block_number BIGINT NOT NULL DEFAULT 0,
		fabric_channel VARCHAR(50) NOT NULL DEFAULT '',
		chaincode_name VARCHAR(50) NOT NULL DEFAULT '',
		payload_json JSON NULL,
		snapshot_json JSON NULL,
		retry_count INT NOT NULL DEFAULT 0,
		confirmed_at DATETIME NULL,
		error_message TEXT NULL,
		event_id VARCHAR(64) NOT NULL DEFAULT '',
		trace_code VARCHAR(32) NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS operation_logs (
		id BIGINT PRIMARY KEY AUTO_INCREMENT, user_id BIGINT NOT NULL, username VARCHAR(50) NOT NULL,
		role VARCHAR(30) NOT NULL, action VARCHAR(100) NOT NULL, detail VARCHAR(255) NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS login_logs (
		id BIGINT PRIMARY KEY AUTO_INCREMENT, user_id BIGINT NULL, username VARCHAR(50) NOT NULL,
		result VARCHAR(30) NOT NULL, ip VARCHAR(64) NOT NULL, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,
}

// fabricMigrations 增量添加 Fabric 字段（对已存在的旧表安全执行）
var fabricMigrations = []string{
	"ALTER TABLE evidence_records DROP INDEX transaction_hash",
	"ALTER TABLE evidence_records DROP INDEX block_hash",
	"ALTER TABLE evidence_records ADD COLUMN fabric_status VARCHAR(20) NOT NULL DEFAULT 'pending'",
	"ALTER TABLE evidence_records ADD COLUMN fabric_tx_id VARCHAR(128) NOT NULL DEFAULT ''",
	"ALTER TABLE evidence_records ADD COLUMN fabric_block_number BIGINT NOT NULL DEFAULT 0",
	"ALTER TABLE evidence_records ADD COLUMN fabric_channel VARCHAR(50) NOT NULL DEFAULT ''",
	"ALTER TABLE evidence_records ADD COLUMN chaincode_name VARCHAR(50) NOT NULL DEFAULT ''",
	"ALTER TABLE evidence_records ADD COLUMN payload_json JSON NULL",
	"ALTER TABLE evidence_records ADD COLUMN snapshot_json JSON NULL",
	"ALTER TABLE evidence_records ADD COLUMN retry_count INT NOT NULL DEFAULT 0",
	"ALTER TABLE evidence_records ADD COLUMN confirmed_at DATETIME NULL",
	"ALTER TABLE evidence_records ADD COLUMN error_message TEXT NULL",
	"ALTER TABLE evidence_records ADD COLUMN event_id VARCHAR(64) NOT NULL DEFAULT ''",
	"ALTER TABLE evidence_records ADD COLUMN trace_code VARCHAR(32) NOT NULL DEFAULT ''",
}

func seedData() error {
	var count int
	if err := DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil || count > 0 {
		return err
	}
	users := []struct{ username, name, role, org, phone, status string }{
		{"admin", "徐文博", "系统管理员", "食用油运输监管平台运维中心（北京市海淀区中关村南大街12号）", "136 0108 5721", "approved"},
		{"supplier", "刘建国", "原料供应商", "黑龙江北沃大豆种植专业合作社（绥化市北林区兴福镇兴福村）", "139 4555 2086", "approved"},
		{"supplier2", "李海峰", "原料供应商", "山东鲁丰花生种植合作社（烟台市莱阳市照旺庄镇西陶漳村）", "138 5352 7619", "approved"},
		{"factory", "周明远", "榨油厂", "中粮佳悦粮油工业有限公司（天津市滨海新区临港经济区渤海路36号）", "137 0206 4188", "approved"},
		{"factory2", "高志远", "榨油厂", "山东鲁油食品工业有限公司（临沂市莒南县经济开发区淮海路88号）", "186 5391 6725", "approved"},
		{"driver", "王志强", "运输人员", "安达恒泰食品运输有限公司（哈尔滨市道外区先锋路469号）", "158 0461 9037", "approved"},
		{"driver2", "马跃", "运输人员", "齐鲁食品物流有限公司（济南市历城区工业北路221号）", "151 6918 4420", "approved"},
		{"retailer", "赵雅琴", "零售商", "北京京粮放心粮油销售有限公司（北京市丰台区新发地食品产业园5号库）", "135 0127 6689", "approved"},
		{"retailer2", "陈晓敏", "零售商", "上海惠民粮油连锁有限公司（上海市浦东新区航头镇航都路18号）", "189 1862 3056", "approved"},
		{"regulator", "陈国栋", "监管机构", "北京市市场监督管理局食品流通处（北京市通州区留庄路6号院）", "010-8269 1732", "approved"},
		{"pending", "孙伟", "运输人员", "顺通食品运输服务有限公司（石家庄市裕华区仓盛路99号）", "177 3118 9054", "pending"},
		{"disabled", "何瑞华", "零售商", "华北优选商贸有限公司（保定市莲池区东风东路278号）", "133 3126 4408", "disabled"},
	}
	for _, u := range users {
		_, err := DB.Exec("INSERT INTO users(username,password,display_name,role,organization,phone,status) VALUES(?,?,?,?,?,?,?)",
			u.username, HashPassword("123456"), u.name, u.role, u.org, u.phone, u.status)
		if err != nil {
			return err
		}
	}
	cases := []seedCase{
		{code: "TR202605200001", material: "非转基因大豆", origin: "黑龙江省绥化市北林区兴福镇兴福村", quantity: 32.80, grade: "一级", report: "北林农检（2026）第0520186号：水分12.4%，杂质0.7%，农残未检出", status: "completed", supplier: "supplier", factory: "factory", driver: "driver", retailer: "retailer", scenario: "corrected"},
		{code: "TR202605230002", material: "高油酸花生仁", origin: "山东省临沂市莒南县大店镇花园村", quantity: 24.60, grade: "一级", report: "莒南质检（2026）第0523072号：酸价0.52mg/g，黄曲霉毒素B1未检出", status: "completed", supplier: "supplier2", factory: "factory2", driver: "driver2", retailer: "retailer2"},
		{code: "TR202605270003", material: "一级葵花籽", origin: "内蒙古自治区巴彦淖尔市临河区白脑包镇", quantity: 27.35, grade: "一级", report: "临河农检（2026）第0527119号：含油率48.6%，水分8.1%", status: "pending_retail", supplier: "supplier", factory: "factory", driver: "driver", retailer: "retailer"},
		{code: "TR202605290004", material: "非转基因大豆", origin: "黑龙江省齐齐哈尔市富裕县友谊乡", quantity: 30.20, grade: "一级", report: "富裕县农检（2026）第0529088号：蛋白质38.7%，水分12.1%", status: "in_transit", supplier: "supplier", factory: "factory", driver: "driver", retailer: "retailer"},
		{code: "TR202605300005", material: "高油酸花生仁", origin: "山东省烟台市莱阳市照旺庄镇西陶漳村", quantity: 19.80, grade: "一级", report: "莱阳农检（2026）第0530045号：酸价0.61mg/g，水分6.3%", status: "in_transit", supplier: "supplier2", factory: "factory2", driver: "driver2", retailer: "retailer2", scenario: "retail_returned"},
		{code: "TR202606010006", material: "非转基因大豆", origin: "黑龙江省佳木斯市桦川县悦来镇", quantity: 28.00, grade: "一级", report: "桦川质检（2026）第0601033号：转基因成分未检出，杂质0.6%", status: "transport_accepted", supplier: "supplier", factory: "factory", driver: "driver", retailer: "retailer"},
		{code: "TR202606020007", material: "一级油菜籽", origin: "湖北省荆州市监利市尺八镇", quantity: 22.45, grade: "一级", report: "监利农检（2026）第0602168号：含油率43.2%，芥酸符合标准", status: "pending_transport", supplier: "supplier", factory: "factory", driver: "driver", retailer: "retailer"},
		{code: "TR202606030008", material: "非转基因大豆", origin: "黑龙江省哈尔滨市双城区永胜镇", quantity: 26.75, grade: "一级", report: "双城农检（2026）第0603114号：水分12.0%，农残未检出", status: "processed", supplier: "supplier", factory: "factory"},
		{code: "TR202606030009", material: "一级花生仁", origin: "山东省青岛市莱西市院上镇", quantity: 18.90, grade: "一级", report: "莱西质检（2026）第0603092号：黄曲霉毒素B1未检出", status: "processed", supplier: "supplier2", factory: "factory2", driver: "driver2", retailer: "retailer2", scenario: "transport_returned"},
		{code: "TR202606040010", material: "非转基因菜籽", origin: "四川省德阳市罗江区鄢家镇", quantity: 21.60, grade: "二级", report: "罗江农检（2026）第0604058号：含油率39.8%，水分7.9%", status: "factory_received", supplier: "supplier", factory: "factory"},
		{code: "TR202606050011", material: "高油酸花生仁", origin: "河南省开封市祥符区仇楼镇", quantity: 17.35, grade: "一级", report: "开封抽检（2026）第0605127号：复检水分9.4%，超出合同接收标准", status: "returned_supplier", supplier: "supplier2", factory: "factory2"},
		{code: "TR202606060012", material: "非转基因大豆", origin: "黑龙江省绥化市兰西县榆林镇", quantity: 29.10, grade: "一级", report: "兰西农检（2026）第0606071号：水分12.2%，杂质0.8%", status: "pending_factory", supplier: "supplier"},
		{code: "TR202606070013", material: "一级花生仁", origin: "山东省烟台市莱阳市万第镇", quantity: 16.80, grade: "一级", report: "待补充正式检验报告", status: "raw_draft", supplier: "supplier2"},
	}
	for index, item := range cases {
		if err := seedBatch(item, time.Now().AddDate(0, 0, -(15-index))); err != nil {
			return err
		}
	}
	return nil
}

type seedCase struct {
	code, material, origin, grade, report, status, supplier, factory, driver, retailer, scenario string
	quantity                                                                                     float64
}

type seedRoute struct {
	startProvince, startCity, endProvince, endCity, vehicle, driverName string
	startLng, startLat, endLng, endLat                                  float64
	nodes                                                               int
	waypoints                                                           []RoutePoint
}

func routeForSeed(code, driver string) seedRoute {
	routes := map[string]seedRoute{
		"TR202605200001": {"天津市", "滨海新区", "北京市", "丰台区", "津C·F6218", "王志强", 117.711913, 38.986438, 116.342820, 39.807650, 12, []RoutePoint{{117.711913, 38.986438}, {117.200983, 39.084158}, {116.838715, 39.726177}, {116.342820, 39.807650}}},
		"TR202605230002": {"山东省", "临沂市", "上海市", "浦东新区", "鲁Q·R5309", "马跃", 118.356448, 35.104672, 121.667180, 31.032920, 18, []RoutePoint{{118.356448, 35.104672}, {119.161755, 34.589050}, {120.311910, 31.491170}, {121.667180, 31.032920}}},
		"TR202605270003": {"天津市", "滨海新区", "北京市", "丰台区", "津C·M7712", "王志强", 117.711913, 38.986438, 116.342820, 39.807650, 10, []RoutePoint{{117.711913, 38.986438}, {117.200983, 39.084158}, {116.342820, 39.807650}}},
		"TR202605290004": {"天津市", "滨海新区", "北京市", "丰台区", "津C·L9086", "王志强", 117.711913, 38.986438, 116.342820, 39.807650, 8, []RoutePoint{{117.711913, 38.986438}, {117.200983, 39.084158}, {116.342820, 39.807650}}},
		"TR202605300005": {"山东省", "临沂市", "上海市", "浦东新区", "鲁Q·V8826", "马跃", 118.356448, 35.104672, 121.667180, 31.032920, 14, []RoutePoint{{118.356448, 35.104672}, {119.161755, 34.589050}, {120.311910, 31.491170}, {121.667180, 31.032920}}},
		"TR202606010006": {"天津市", "滨海新区", "北京市", "丰台区", "津C·A6621", "王志强", 117.711913, 38.986438, 116.342820, 39.807650, 0, []RoutePoint{{117.711913, 38.986438}, {116.342820, 39.807650}}},
		"TR202606020007": {"天津市", "滨海新区", "北京市", "丰台区", "津C·QH218", "王志强", 117.711913, 38.986438, 116.342820, 39.807650, 0, []RoutePoint{{117.711913, 38.986438}, {116.342820, 39.807650}}},
		"TR202606030009": {"山东省", "临沂市", "上海市", "浦东新区", "鲁Q·K1735", "马跃", 118.356448, 35.104672, 121.667180, 31.032920, 0, []RoutePoint{{118.356448, 35.104672}, {121.667180, 31.032920}}},
	}
	route, ok := routes[code]
	if !ok {
		driverName := map[string]string{"driver": "王志强", "driver2": "马跃"}[driver]
		route = seedRoute{"天津市", "滨海新区", "北京市", "丰台区", "津C·S6621", driverName, 117.711913, 38.986438, 116.342820, 39.807650, 10, DomesticRoute("滨海新区", "丰台区", 117.711913, 38.986438, 116.342820, 39.807650)}
	}
	return route
}

type RoutePoint struct {
	Lng float64
	Lat float64
}

func DomesticRoute(startCity, endCity string, startLng, startLat, endLng, endLat float64) []RoutePoint {
	key := startCity + "->" + endCity
	routes := map[string][]RoutePoint{
		"秦皇岛市->北京市": {{startLng, startLat}, {119.180201, 39.630680}, {117.200983, 39.084158}, {116.407526, 39.904030}},
		"哈尔滨市->北京市": {{startLng, startLat}, {125.323544, 43.817071}, {123.431474, 41.805698}, {121.127003, 41.095119}, {119.600492, 39.935385}, {118.180149, 39.630680}, {endLng, endLat}},
	}
	if route, ok := routes[key]; ok {
		route[len(route)-1] = RoutePoint{endLng, endLat}
		return route
	}
	return []RoutePoint{{startLng, startLat}, {endLng, endLat}}
}

func SampleRoute(route []RoutePoint, count int) []RoutePoint {
	if count <= 0 || len(route) < 2 {
		return nil
	}
	lengths := make([]float64, len(route)-1)
	total := 0.0
	for i := 0; i < len(route)-1; i++ {
		dx, dy := route[i+1].Lng-route[i].Lng, route[i+1].Lat-route[i].Lat
		lengths[i] = math.Sqrt(dx*dx + dy*dy)
		total += lengths[i]
	}
	points := make([]RoutePoint, 0, count)
	for i := 1; i <= count; i++ {
		target := total * float64(i) / float64(count+1)
		traveled := 0.0
		for j, length := range lengths {
			if target <= traveled+length || j == len(lengths)-1 {
				ratio := (target - traveled) / length
				points = append(points, RoutePoint{
					Lng: route[j].Lng + (route[j+1].Lng-route[j].Lng)*ratio,
					Lat: route[j].Lat + (route[j+1].Lat-route[j].Lat)*ratio,
				})
				break
			}
			traveled += length
		}
	}
	return points
}

func seedBatch(item seedCase, base time.Time) error {
	ids := map[string]int64{}
	for _, name := range []string{item.supplier, item.factory, item.driver, item.retailer} {
		if name == "" {
			continue
		}
		var id int64
		if err := DB.QueryRow("SELECT id FROM users WHERE username=?", name).Scan(&id); err != nil {
			return err
		}
		ids[name] = id
	}
	productName := "一级压榨大豆油"
	if strings.Contains(item.material, "花生") {
		productName = "一级压榨花生油"
	} else if strings.Contains(item.material, "菜籽") || strings.Contains(item.material, "油菜") {
		productName = "一级压榨菜籽油"
	} else if strings.Contains(item.material, "葵花") {
		productName = "一级压榨葵花籽油"
	}
	hasProcessing := map[string]bool{"processed": true, "pending_transport": true, "transport_accepted": true, "in_transit": true, "pending_retail": true, "completed": true}[item.status]
	processing := []byte("null")
	if hasProcessing {
		processing, _ = json.Marshal(map[string]interface{}{"product_name": productName, "process": "原料清理、磁选、低温压榨、物理精炼、氮气保护灌装", "production_batch": "SC-" + item.code[2:] + "-01", "production_time": base.Add(48 * time.Hour).Format("2006-01-02 15:04:05"), "inspection_report": "成品检验报告 CY-" + item.code[8:] + "：酸价、过氧化值、溶剂残留量均符合 GB/T 1535 要求", "quality_manager": "林雪梅"})
	}
	receipt := []byte("null")
	if item.status == "completed" {
		receipt, _ = json.Marshal(map[string]interface{}{"result": "确认收货", "quantity": math.Round(item.quantity*0.91*100) / 100, "received_time": base.Add(120 * time.Hour).Format("2006-01-02 15:04:05"), "quality": "罐体及铅封完整，随车检验报告与批次一致，抽检感官指标合格", "warehouse": "新发地食品产业园5号食品级成品油库", "receiver": "赵雅琴"})
	}
	var factoryID, driverID, retailerID interface{}
	if item.factory != "" {
		factoryID = ids[item.factory]
	}
	if item.driver != "" {
		driverID = ids[item.driver]
	}
	if item.retailer != "" {
		retailerID = ids[item.retailer]
	}
	res, err := DB.Exec(`INSERT INTO batches(trace_code,supplier_id,oil_factory_id,transporter_id,retailer_id,status,material_name,origin,quantity,unit,quality_grade,production_date,test_report,processing_data,receipt_data)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, item.code, ids[item.supplier], factoryID, driverID, retailerID, item.status, item.material, item.origin, item.quantity, "吨", item.grade, base.Add(-24*time.Hour).Format("2006-01-02"), item.report, processing, receipt)
	if err != nil {
		return err
	}
	batchID, _ := res.LastInsertId()
	_, _ = DB.Exec("UPDATE batches SET created_at=?,updated_at=? WHERE id=?", base, base.Add(2*time.Hour), batchID)

	if item.status != "raw_draft" {
		_ = CreateEvidenceAt(batchID, "原料批次存证", "原料供应商提交原料批次信息", ids[item.supplier], "原料供应商", base.Add(2*time.Hour))
	}
	if item.status == "returned_supplier" {
		_, _ = DB.Exec("INSERT INTO rejection_records(batch_id,stage,reason,operator_id,created_at) VALUES(?,'原料接收',?,?,?)", batchID, "抽样检测水分含量超出接收标准", ids["factory"], base.Add(12*time.Hour))
		_ = CreateEvidenceAt(batchID, "原料拒收存证", "榨油厂拒收原料：抽样检测水分含量超出接收标准", ids["factory"], "榨油厂", base.Add(12*time.Hour))
		return nil
	}
	if item.factory == "" {
		return nil
	}
	_ = CreateEvidenceAt(batchID, "原料接收存证", "榨油厂确认接收原料", ids[item.factory], "榨油厂", base.Add(24*time.Hour))
	if !hasProcessing {
		return nil
	}
	_ = CreateEvidenceAt(batchID, "加工生产存证", "榨油厂提交加工生产信息", ids[item.factory], "榨油厂", base.Add(48*time.Hour))
	if item.driver == "" || item.retailer == "" || (item.status == "processed" && item.scenario != "transport_returned") {
		return nil
	}
	route := routeForSeed(item.code, item.driver)
	taskStatus := map[string]string{"pending_transport": "pending_accept", "transport_accepted": "accepted"}[item.status]
	if taskStatus == "" {
		taskStatus = item.status
	}
	var startedAt, completedAt interface{}
	if item.status == "in_transit" || item.status == "pending_retail" || item.status == "completed" {
		startedAt = base.Add(72 * time.Hour)
	}
	if item.status == "pending_retail" || item.status == "completed" {
		completedAt = base.Add(108 * time.Hour)
	}
	res, err = DB.Exec(`INSERT INTO transport_tasks(batch_id,factory_id,transporter_id,retailer_id,vehicle_no,driver_name,product_name,product_quantity,start_province,start_city,start_lng,start_lat,end_province,end_city,end_lng,end_lat,status,note,created_at,started_at,completed_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		batchID, ids[item.factory], ids[item.driver], ids[item.retailer], route.vehicle, route.driverName, productName, math.Round(item.quantity*0.91*100)/100,
		route.startProvince, route.startCity, route.startLng, route.startLat, route.endProvince, route.endCity, route.endLng, route.endLat, taskStatus,
		fmt.Sprintf("国内食品级专用罐车运输，%s至%s，全程铅封并记录定位及温湿度", route.startCity, route.endCity), base.Add(60*time.Hour), startedAt, completedAt)
	if err != nil {
		return err
	}
	taskID, _ := res.LastInsertId()
	_ = CreateEvidenceAt(batchID, "运输任务存证", "榨油厂发起运输任务", ids[item.factory], "榨油厂", base.Add(60*time.Hour))
	if item.scenario == "transport_returned" {
		reason := "承运车辆食品级罐体清洗证明已过有效期，运输人员退回任务"
		_, _ = DB.Exec("INSERT INTO rejection_records(batch_id,stage,reason,operator_id,created_at) VALUES(?,'运输任务',?,?,?)", batchID, reason, ids[item.driver], base.Add(64*time.Hour))
		_, _ = DB.Exec("DELETE FROM transport_tasks WHERE id=?", taskID)
		_ = CreateEvidenceAt(batchID, "运输任务退回存证", "运输人员退回运输任务："+reason, ids[item.driver], "运输人员", base.Add(64*time.Hour))
		return nil
	}
	if item.status == "pending_transport" {
		return nil
	}
	_ = CreateEvidenceAt(batchID, "运输接收存证", "运输人员确认接收任务", ids[item.driver], "运输人员", base.Add(66*time.Hour))
	if item.status == "transport_accepted" {
		return nil
	}
	_ = CreateEvidenceAt(batchID, "运输启运存证", "运输人员开始运输", ids[item.driver], "运输人员", base.Add(72*time.Hour))
	for i, point := range SampleRoute(route.waypoints, route.nodes) {
		_, _ = DB.Exec("INSERT INTO transport_nodes(task_id,seq,longitude,latitude,temperature,humidity,recorded_at) VALUES(?,?,?,?,?,?,?)",
			taskID, i+1, point.Lng, point.Lat, 17.8+math.Sin(float64(i))*1.6, 49.0+math.Cos(float64(i))*4.0, base.Add(time.Duration(74+i*2)*time.Hour))
	}
	_ = CreateEvidenceAt(batchID, "运输过程存证", "运输人员上传运输轨迹及温湿度数据", ids[item.driver], "运输人员", base.Add(96*time.Hour))
	if item.scenario == "retail_returned" {
		_, _ = DB.Exec("UPDATE transport_tasks SET completed_at=? WHERE id=?", base.Add(108*time.Hour), taskID)
		_ = CreateEvidenceAt(batchID, "运输完成存证", "运输任务完成，等待零售商收货", ids[item.driver], "运输人员", base.Add(108*time.Hour))
		reason := "到货复核发现随车纸质检验报告缺少骑缝章，退回运输人员补正"
		_, _ = DB.Exec("INSERT INTO rejection_records(batch_id,stage,reason,operator_id,created_at) VALUES(?,'零售收货',?,?,?)", batchID, reason, ids[item.retailer], base.Add(112*time.Hour))
		_, _ = DB.Exec("UPDATE transport_tasks SET completed_at=NULL WHERE id=?", taskID)
		_ = CreateEvidenceAt(batchID, "零售退回存证", "零售商退回产品："+reason, ids[item.retailer], "零售商", base.Add(113*time.Hour))
		return nil
	}
	if item.status == "in_transit" {
		return nil
	}
	_ = CreateEvidenceAt(batchID, "运输完成存证", "运输任务完成，等待零售商收货", ids[item.driver], "运输人员", base.Add(108*time.Hour))
	if item.status == "completed" {
		_ = CreateEvidenceAt(batchID, "零售收货存证", "零售商确认产品收货并填写入库核验信息", ids[item.retailer], "零售商", base.Add(120*time.Hour))
	}
	if item.scenario == "corrected" {
		content := "补充原料采购合同编号：CG-2026-BW-0520，并说明不改变原始检测及运输数据"
		_, _ = DB.Exec("INSERT INTO corrections(batch_id,stage,content,reason,operator_id,created_at) VALUES(?,'原料采购',?,?,?,?)", batchID, content, "归档复核时发现采购合同编号漏填", ids[item.supplier], base.Add(126*time.Hour))
		_ = CreateEvidenceAt(batchID, "数据更正存证", "原料采购追加更正：归档复核时发现采购合同编号漏填", ids[item.supplier], "原料供应商", base.Add(126*time.Hour))
	}
	return nil
}

func HashPassword(password string) string {
	sum := sha256.Sum256([]byte("oil-supervision:" + password))
	return hex.EncodeToString(sum[:])
}

func GenerateTraceCode() string {
	return "TR" + time.Now().Format("20060102150405") + randomHex(3)
}

func randomHex(n int) string {
	buf := make([]byte, n)
	_, _ = cryptorand.Read(buf)
	return strings.ToUpper(hex.EncodeToString(buf))
}

func CreateEvidence(batchID int64, businessType, summary string, operatorID int64, role string) error {
	return CreateEvidenceAt(batchID, businessType, summary, operatorID, role, time.Now())
}

func CreateEvidenceAt(batchID int64, businessType, summary string, operatorID int64, role string, createdAt time.Time) error {
	// 1. 查找该批次的溯源码
	var traceCode string
	if batchID > 0 {
		_ = DB.QueryRow("SELECT trace_code FROM batches WHERE id=?", batchID).Scan(&traceCode)
	}

	// 2. Build a deterministic snapshot of the complete SQL business data for
	// this stage. The snapshot stays in MySQL; only its SHA-256 hash is on-chain.
	snapshot, err := BuildEvidenceSnapshot(batchID, businessType, summary, operatorID, role, createdAt)
	if err != nil {
		return err
	}
	dataHash := blockchain.ComputePayloadHash(snapshot)
	snapshotJSON, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}

	// 3. 构建链上事件
	offchainRef := fmt.Sprintf("batches:%d", batchID)
	chainEvent := blockchain.BuildChainEvent(businessType, summary, dataHash, operatorID, role, traceCode, offchainRef)
	eventJSON, _ := json.Marshal(chainEvent)

	// 4. 仍然保留兼容哈希字段
	previous := strings.Repeat("0", 64)
	_ = DB.QueryRow("SELECT COALESCE(data_hash,'') FROM evidence_records WHERE batch_id=? ORDER BY id DESC LIMIT 1", batchID).Scan(&previous)
	if previous == "" {
		previous = strings.Repeat("0", 64)
	}

	// 5. 写入 MySQL，状态为 pending
	res, err := DB.Exec(`INSERT INTO evidence_records(batch_id,business_type,business_summary,data_hash,previous_hash,
		transaction_hash,block_hash,operator_id,operator_role,fabric_status,payload_json,snapshot_json,event_id,trace_code,created_at)
		VALUES(?,?,?,?,?,?,?,?,?,'pending',?,?,?,?,?)`,
		nullableBatch(batchID), businessType, summary, dataHash, previous,
		"", "", operatorID, role, string(eventJSON), string(snapshotJSON), chainEvent.EventID, traceCode, createdAt)
	if err != nil {
		return err
	}
	recordID, _ := res.LastInsertId()

	// 6. 异步提交 Fabric 交易
	go func() {
		if !blockchain.GetGateway().IsEnabled() {
			log.Printf("[Fabric] 离线模式，记录 %d 保持 pending 状态", recordID)
			return
		}

		result, err := blockchain.SubmitEvent(chainEvent, role)
		if err != nil {
			log.Printf("[Fabric] 记录 %d 上链失败: %v", recordID, err)
			DB.Exec(`UPDATE evidence_records SET fabric_status='failed',
				error_message=?, retry_count=1 WHERE id=?`, err.Error(), recordID)
			return
		}

		// 上链成功，更新记录
		DB.Exec(`UPDATE evidence_records SET fabric_status='confirmed',
			fabric_tx_id=?, fabric_block_number=?,
			fabric_channel=?, chaincode_name=?,
			transaction_hash=?, confirmed_at=NOW(), error_message=NULL WHERE id=?`,
			result.TxID, result.BlockNumber,
			"oiltracechannel", "oiltrace",
			result.TxID, recordID)
		log.Printf("[Fabric] 记录 %d 上链成功: txID=%s, block=%d", recordID, result.TxID, result.BlockNumber)
	}()

	return nil
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func nullableBatch(id int64) interface{} {
	if id == 0 {
		return nil
	}
	return id
}

func GenToken(userID int64, username, role string) (string, error) {
	claims := Claims{UserID: userID, Username: username, Role: role, StandardClaims: jwt.StandardClaims{
		ExpiresAt: time.Now().Add(TokenExpireDuration).Unix(), Issuer: "oil-supervision",
	}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(viper.GetString("jwt.secret")))
}

func ParseToken(tokenString string) (*Claims, error) {
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("invalid signing method")
		}
		return []byte(viper.GetString("jwt.secret")), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

func Log(userID int64, username, role, action, detail string) {
	_, _ = DB.Exec("INSERT INTO operation_logs(user_id,username,role,action,detail) VALUES(?,?,?,?,?)", userID, username, role, action, detail)
}
