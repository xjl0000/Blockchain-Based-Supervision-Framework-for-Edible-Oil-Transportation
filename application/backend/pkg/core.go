package pkg

import (
	cryptorand "crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
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
		business_summary VARCHAR(255) NOT NULL, data_hash VARCHAR(64) NOT NULL, previous_hash VARCHAR(64) NOT NULL,
		transaction_hash VARCHAR(64) UNIQUE NOT NULL, block_hash VARCHAR(64) UNIQUE NOT NULL,
		operator_id BIGINT NOT NULL, operator_role VARCHAR(30) NOT NULL,
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

func seedData() error {
	var count int
	if err := DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil || count > 0 {
		return err
	}
	users := []struct{ username, name, role, org string }{
		{"admin", "徐文博", "系统管理员", "食用油运输监管平台运维中心"},
		{"supplier", "刘建国", "原料供应商", "黑龙江北沃大豆种植专业合作社"},
		{"supplier2", "李海峰", "原料供应商", "山东鲁丰花生种植合作社"},
		{"factory", "周明远", "榨油厂", "中粮佳悦粮油工业有限公司"},
		{"factory2", "高志远", "榨油厂", "山东鲁油食品工业有限公司"},
		{"driver", "王志强", "运输人员", "安达恒泰食品运输有限公司"},
		{"driver2", "马跃", "运输人员", "齐鲁食品物流有限公司"},
		{"retailer", "赵雅琴", "零售商", "北京京粮放心粮油销售有限公司"},
		{"retailer2", "陈晓敏", "零售商", "上海惠民粮油连锁有限公司"},
		{"regulator", "陈国栋", "监管机构", "北京市市场监督管理局食品流通处"},
		{"pending", "孙伟", "运输人员", "顺通食品运输服务有限公司"},
	}
	for _, u := range users {
		status := "approved"
		if u.username == "pending" {
			status = "pending"
		}
		_, err := DB.Exec("INSERT INTO users(username,password,display_name,role,organization,phone,status) VALUES(?,?,?,?,?,?,?)",
			u.username, HashPassword("123456"), u.name, u.role, u.org, "13800000000", status)
		if err != nil {
			return err
		}
	}
	cases := []seedCase{
		{code: "TR202606010001", material: "非转基因大豆", origin: "黑龙江省绥化市北林区", status: "completed", supplier: "supplier", factory: "factory", driver: "driver", retailer: "retailer"},
		{code: "TR202606020002", material: "一级压榨花生仁", origin: "山东省青岛市莱西市", status: "in_transit", supplier: "supplier", factory: "factory", driver: "driver", retailer: "retailer"},
		{code: "TR202606030003", material: "非转基因菜籽", origin: "四川省德阳市罗江区", status: "pending_factory", supplier: "supplier"},
		{code: "TR202606030004", material: "高油酸花生仁", origin: "河南省开封市祥符区", status: "returned_supplier", supplier: "supplier", factory: "factory"},
		{code: "TR202606040005", material: "非转基因大豆", origin: "黑龙江省佳木斯市桦川县", status: "factory_received", supplier: "supplier", factory: "factory"},
		{code: "TR202606040006", material: "一级油菜籽", origin: "湖北省荆州市监利市", status: "processed", supplier: "supplier", factory: "factory"},
		{code: "TR202606050007", material: "非转基因大豆", origin: "黑龙江省齐齐哈尔市", status: "pending_transport", supplier: "supplier", factory: "factory", driver: "driver", retailer: "retailer"},
		{code: "TR202606050008", material: "一级葵花籽", origin: "内蒙古自治区巴彦淖尔市", status: "pending_retail", supplier: "supplier", factory: "factory", driver: "driver", retailer: "retailer"},
		{code: "TR202605280009", material: "高油酸花生仁", origin: "山东省临沂市莒南县", status: "completed", supplier: "supplier2", factory: "factory2", driver: "driver2", retailer: "retailer2"},
		{code: "TR202606060010", material: "一级花生仁", origin: "山东省烟台市莱阳市", status: "raw_draft", supplier: "supplier2"},
	}
	for index, item := range cases {
		if err := seedBatch(item, time.Now().AddDate(0, 0, -(15-index))); err != nil {
			return err
		}
	}
	return nil
}

type seedCase struct {
	code, material, origin, status, supplier, factory, driver, retailer string
}

type seedRoute struct {
	startProvince, startCity, endProvince, endCity, vehicle, driverName string
	startLng, startLat, endLng, endLat                                  float64
	nodes                                                               int
	waypoints                                                           []RoutePoint
}

func routeForSeed(code, driver string) seedRoute {
	routes := map[string]seedRoute{
		"TR202606010001": {"黑龙江省", "哈尔滨市", "北京市", "北京市", "黑A·E6608", "王志强", 126.642464, 45.756967, 116.407526, 39.904030, 18, []RoutePoint{{126.642464, 45.756967}, {125.323544, 43.817071}, {124.350398, 43.166419}, {123.431474, 41.805698}, {121.127003, 41.095119}, {119.600492, 39.935385}, {118.180149, 39.630680}, {116.407526, 39.904030}}},
		"TR202606020002": {"河北省", "秦皇岛市海港区", "河北省", "秦皇岛市山海关区", "冀C·QH218", "王志强", 119.600492, 39.935385, 119.775799, 39.978848, 10, []RoutePoint{{119.600492, 39.935385}, {119.637454, 39.942190}, {119.681104, 39.950858}, {119.727870, 39.966231}, {119.775799, 39.978848}}},
		"TR202606050007": {"黑龙江省", "齐齐哈尔市", "广东省", "广州市", "黑B·L9086", "王志强", 123.918186, 47.354348, 113.264385, 23.129112, 24, []RoutePoint{{123.918186, 47.354348}, {125.103784, 46.589309}, {126.642464, 45.756967}, {125.323544, 43.817071}, {123.431474, 41.805698}, {117.200983, 39.084158}, {114.514859, 38.042306}, {113.625368, 34.746599}, {114.305392, 30.593098}, {112.938814, 28.228209}, {113.597522, 24.810403}, {113.264385, 23.129112}}},
		"TR202606050008": {"内蒙古自治区", "巴彦淖尔市", "北京市", "北京市", "蒙L·Y7712", "王志强", 107.387657, 40.743213, 116.407526, 39.904030, 16, []RoutePoint{{107.387657, 40.743213}, {109.840347, 40.657449}, {111.749180, 40.842585}, {114.885895, 40.768931}, {116.407526, 39.904030}}},
		"TR202605280009": {"山东省", "临沂市", "山东省", "青岛市", "鲁Q·F5309", "马跃", 118.356448, 35.104672, 120.382640, 36.067082, 12, []RoutePoint{{118.356448, 35.104672}, {119.526888, 35.416377}, {119.995518, 35.875138}, {120.197353, 35.960688}, {120.382640, 36.067082}}},
	}
	route, ok := routes[code]
	if !ok {
		route = seedRoute{"河北省", "秦皇岛市", "北京市", "北京市", "冀C·S6621", driver, 119.520220, 39.888243, 116.407526, 39.904030, 14, DomesticRoute("秦皇岛市", "北京市", 119.520220, 39.888243, 116.407526, 39.904030)}
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
		processing, _ = json.Marshal(map[string]interface{}{"product_name": productName, "process": "原料筛选、低温压榨、物理精炼、灌装封装", "production_batch": item.code + "-P", "production_time": base.Add(48 * time.Hour).Format("2006-01-02 15:04:05"), "inspection": "酸价、过氧化值、溶剂残留量检验合格"})
	}
	receipt := []byte("null")
	if item.status == "completed" {
		receipt, _ = json.Marshal(map[string]interface{}{"result": "确认收货", "quantity": 18.6, "received_time": base.Add(120 * time.Hour).Format("2006-01-02 15:04:05"), "quality": "铅封完整，包装无破损，随车检验报告核验通过", "warehouse": "食品级成品油专用仓"})
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
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, item.code, ids[item.supplier], factoryID, driverID, retailerID, item.status, item.material, item.origin, 20, "吨", "一级", base.Add(-24*time.Hour).Format("2006-01-02"), "农残、酸价及水分检测合格", processing, receipt)
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
	if item.driver == "" || item.retailer == "" || item.status == "processed" {
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
		batchID, ids[item.factory], ids[item.driver], ids[item.retailer], route.vehicle, route.driverName, productName, 18.6,
		route.startProvince, route.startCity, route.startLng, route.startLat, route.endProvince, route.endCity, route.endLng, route.endLat, taskStatus,
		fmt.Sprintf("国内食品级专用罐车运输，%s至%s，全程铅封并记录定位及温湿度", route.startCity, route.endCity), base.Add(60*time.Hour), startedAt, completedAt)
	if err != nil {
		return err
	}
	taskID, _ := res.LastInsertId()
	_ = CreateEvidenceAt(batchID, "运输任务存证", "榨油厂发起运输任务", ids[item.factory], "榨油厂", base.Add(60*time.Hour))
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
	if item.status == "in_transit" {
		return nil
	}
	_ = CreateEvidenceAt(batchID, "运输完成存证", "运输任务完成，等待零售商收货", ids[item.driver], "运输人员", base.Add(108*time.Hour))
	if item.status == "completed" {
		_ = CreateEvidenceAt(batchID, "零售收货存证", "零售商确认产品收货并填写入库核验信息", ids[item.retailer], "零售商", base.Add(120*time.Hour))
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
	payload := fmt.Sprintf("%d|%s|%s|%d|%d", batchID, businessType, summary, operatorID, createdAt.UnixNano())
	dataHash := sha256Hex(payload)
	previous := strings.Repeat("0", 64)
	_ = DB.QueryRow("SELECT block_hash FROM evidence_records WHERE batch_id=? ORDER BY id DESC LIMIT 1", batchID).Scan(&previous)
	txHash := sha256Hex("TX|" + dataHash + "|" + randomHex(8))
	blockHash := sha256Hex(previous + "|" + txHash + "|" + strconv.FormatInt(createdAt.UnixNano(), 10))
	_, err := DB.Exec(`INSERT INTO evidence_records(batch_id,business_type,business_summary,data_hash,previous_hash,transaction_hash,block_hash,operator_id,operator_role,created_at)
		VALUES(?,?,?,?,?,?,?,?,?,?)`, nullableBatch(batchID), businessType, summary, dataHash, previous, txHash, blockHash, operatorID, role, createdAt)
	return err
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
