package controller

import (
	"backend/pkg"
	"database/sql"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

var publicRoles = map[string]bool{"原料供应商": true, "榨油厂": true, "运输人员": true, "零售商": true, "监管机构": true}

func Register(c *gin.Context) {
	var req struct {
		Username, Password, DisplayName, Role, Organization, Phone string
	}
	if c.ShouldBindJSON(&req) != nil || !publicRoles[req.Role] || strings.TrimSpace(req.Username) == "" || len(req.Password) < 6 {
		fail(c, 400, "请完整填写注册信息，密码至少6位")
		return
	}
	_, err := pkg.DB.Exec(`INSERT INTO users(username,password,display_name,role,organization,phone,status) VALUES(?,?,?,?,?,?,'pending')`,
		req.Username, pkg.HashPassword(req.Password), req.DisplayName, req.Role, req.Organization, req.Phone)
	if err != nil {
		fail(c, 400, "用户名已存在或注册信息无效")
		return
	}
	ok(c, gin.H{"message": "注册申请已提交，请等待系统管理员审核"})
}

func Login(c *gin.Context) {
	var req struct{ Username, Password string }
	_ = c.ShouldBindJSON(&req)
	var id int64
	var password, name, role, status, org string
	err := pkg.DB.QueryRow("SELECT id,password,display_name,role,status,organization FROM users WHERE username=?", req.Username).
		Scan(&id, &password, &name, &role, &status, &org)
	if err != nil || password != pkg.HashPassword(req.Password) {
		_, _ = pkg.DB.Exec("INSERT INTO login_logs(username,result,ip) VALUES(?,'登录失败',?)", req.Username, c.ClientIP())
		fail(c, 401, "账号或密码错误")
		return
	}
	if status != "approved" {
		msg := map[string]string{"pending": "账号正在等待管理员审核", "rejected": "注册申请已被拒绝", "disabled": "账号已被禁用"}[status]
		fail(c, 403, msg)
		return
	}
	token, _ := pkg.GenToken(id, req.Username, role)
	_, _ = pkg.DB.Exec("INSERT INTO login_logs(user_id,username,result,ip) VALUES(?,?,'登录成功',?)", id, req.Username, c.ClientIP())
	ok(c, gin.H{"message": "登录成功", "jwt": token, "user": gin.H{"id": id, "username": req.Username, "name": name, "role": role, "organization": org}})
}

func GetInfo(c *gin.Context) {
	id := userID(c)
	var username, name, role, org, phone, status string
	err := pkg.DB.QueryRow("SELECT username,display_name,role,organization,phone,status FROM users WHERE id=?", id).
		Scan(&username, &name, &role, &org, &phone, &status)
	if err != nil {
		fail(c, 404, "用户不存在")
		return
	}
	ok(c, gin.H{"username": username, "name": name, "role": role, "userType": role, "organization": org, "phone": phone, "status": status})
}

func Logout(c *gin.Context) { ok(c, gin.H{"message": "退出成功"}) }

func ListUsers(c *gin.Context) {
	rows, err := pkg.DB.Query("SELECT id,username,display_name,role,organization,phone,status,review_note,created_at FROM users ORDER BY FIELD(status,'pending','approved','disabled','rejected'),id")
	if err != nil {
		fail(c, 500, err.Error())
		return
	}
	defer rows.Close()
	data := []gin.H{}
	for rows.Next() {
		var id int64
		var username, name, role, org, phone, status, note string
		var created interface{}
		_ = rows.Scan(&id, &username, &name, &role, &org, &phone, &status, &note, &created)
		data = append(data, gin.H{"id": id, "username": username, "display_name": name, "role": role, "organization": org, "phone": phone, "status": status, "review_note": note, "created_at": created})
	}
	ok(c, gin.H{"data": data})
}

func UpdateUserStatus(c *gin.Context) {
	var req struct {
		ID     int64  `json:"id"`
		Status string `json:"status"`
		Note   string `json:"note"`
	}
	_ = c.ShouldBindJSON(&req)
	allowed := map[string]bool{"approved": true, "rejected": true, "disabled": true}
	if !allowed[req.Status] {
		fail(c, 400, "无效账号状态")
		return
	}
	var role string
	if pkg.DB.QueryRow("SELECT role FROM users WHERE id=?", req.ID).Scan(&role) != nil || role == "系统管理员" {
		fail(c, 400, "系统管理员账号不可修改")
		return
	}
	_, err := pkg.DB.Exec("UPDATE users SET status=?,review_note=? WHERE id=?", req.Status, req.Note, req.ID)
	if err != nil {
		fail(c, 500, err.Error())
		return
	}
	logAction(c, "账号状态管理", "更新用户账号状态为"+req.Status)
	ok(c, gin.H{"message": "账号状态已更新"})
}

func RoleOptions(c *gin.Context) {
	role := c.Query("role")
	rows, err := pkg.DB.Query("SELECT id,display_name,organization FROM users WHERE role=? AND status='approved' ORDER BY id", role)
	if err != nil {
		fail(c, 500, err.Error())
		return
	}
	defer rows.Close()
	data := []gin.H{}
	for rows.Next() {
		var id int64
		var name, org string
		_ = rows.Scan(&id, &name, &org)
		data = append(data, gin.H{"id": id, "name": name, "organization": org})
	}
	ok(c, gin.H{"data": data})
}

func Logs(c *gin.Context) {
	kind := c.DefaultQuery("kind", "operation")
	query := "SELECT username,role,action,detail,created_at FROM operation_logs ORDER BY id DESC LIMIT 200"
	if kind == "login" {
		query = "SELECT username,'' role,result action,ip detail,created_at FROM login_logs ORDER BY id DESC LIMIT 200"
	}
	rows, err := pkg.DB.Query(query)
	if err != nil {
		fail(c, 500, err.Error())
		return
	}
	defer rows.Close()
	data := []gin.H{}
	for rows.Next() {
		var username, role, action, detail string
		var created interface{}
		_ = rows.Scan(&username, &role, &action, &detail, &created)
		data = append(data, gin.H{"username": username, "role": role, "action": action, "detail": detail, "created_at": created})
	}
	ok(c, gin.H{"data": data})
}

func userID(c *gin.Context) int64 {
	v, _ := c.Get("userID")
	return v.(int64)
}
func role(c *gin.Context) string     { v, _ := c.Get("role"); return v.(string) }
func username(c *gin.Context) string { v, _ := c.Get("username"); return v.(string) }
func logAction(c *gin.Context, action, detail string) {
	pkg.Log(userID(c), username(c), role(c), action, detail)
}
func ok(c *gin.Context, data gin.H) { data["code"] = 200; c.JSON(http.StatusOK, data) }
func fail(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"code": status, "message": message})
}
func nullableInt(value sql.NullInt64) interface{} {
	if value.Valid {
		return value.Int64
	}
	return nil
}
