# 源码阅读与学习规划

> 目标：以“弄懂项目如何运转”为准，不逐行深挖所有细节。优先读入口、主流程、状态流转、模块边界和关键数据表。
>
> 当前项目可以理解为：**食用油运输监管业务系统**，由 Go + Gin 后端、Vue2 前端、MySQL 本地业务数据与存证表、Hyperledger Fabric 链码/网络脚本组成。需要特别注意：当前应用后端的主业务存证主要通过 `application/backend/pkg/core.go` 中的 `CreateEvidenceAt` 写入 MySQL 的 `evidence_records` 表，并用 `previous_hash / transaction_hash / block_hash` 形成本地哈希链；`blockchain/` 下的 Fabric 链码与网络是独立的区块链实现与部署基础，不是 Go 后端当前请求链路中的直接调用依赖。

---

## 一、推荐阅读顺序

1. 先读启动与路由：`start-local.ps1`、`docker-compose.yml`、`application/backend/main.go`、`application/backend/router/router.go`。
2. 再读权限与用户：`middleware/auth.go`、`controller/user.go`、`pkg/core.go` 中 JWT、用户初始化、日志相关函数。
3. 重点读业务主流程：`controller/business.go`，只跟状态变化和 `pkg.CreateEvidence` 调用点，不追每个字段的 UI 展示细节。
4. 读前端调用链：`src/utils/request.js`、`src/api/business.js`、`src/router/index.js`、几个核心页面 `batches / pressing / transport / retail / trace / evidence`。
5. 最后读 Fabric 链码和网络脚本：`blockchain/chaincode/trace.go`、`chaincode/model.go`、`chaincode/smartcontract.go`、`blockchain/network/network.sh`。

---

# 第一阶段：项目模块划分

## 模块 1：启动、配置与本地运行环境

### 1. 前置语法补充

你已有 Python/C++ 基础，补这些即可：

- Go：`package main`、`import`、`func main()`、多返回值、`if err != nil` 错误处理。
- Go 包管理：`go.mod` 中 `module / require` 的含义。
- YAML：缩进式配置，读取配置项时对应 `viper.GetString("mysql.host")` 这类路径。
- PowerShell / Docker Compose 基础：变量、命令执行、容器服务、端口映射、健康检查。

### 2. 核心源码突破

| 文件 | 重点对象 | 作用 | 阅读目标 |
|---|---|---|---|
| `start-local.ps1` | 脚本整体 | 本地一键启动：检查 Docker / Go / npm，启动 MySQL，构建前端，复制前端产物到后端 `dist`，编译并启动 Go 后端 | 明白开发环境是怎么起来的 |
| `docker-compose.yml` | `mysql` service | 启动 MySQL 8，映射宿主机 `3337` 到容器 `3306` | 明白后端数据源在哪里 |
| `application/backend/settings/config.yaml` | `app / jwt / mysql` | 配置后端端口、JWT 密钥、MySQL 连接参数 | 后续读数据库初始化时对上配置 |
| `application/backend/settings/settings.go` | `Init()` | 用 Viper 加载 `settings/config.yaml` | 明白配置加载入口 |
| `application/backend/main.go` | `main()` | 程序启动入口：加载配置、初始化 DB、启动 Gin 路由 | 这是后端启动主线 |
| `application/backend/go.mod` | `require` | Gin、MySQL driver、JWT、Viper 依赖 | 知道项目主要技术栈 |

### 阅读提示

本模块不用深读 PowerShell 每个分支。只要能画出这条线即可：

```text
start-local.ps1
  -> docker compose up mysql
  -> npm run build:prod
  -> copy web/dist 到 backend/dist
  -> go build backend
  -> 启动 oil-supervision-server.exe
  -> 浏览器访问 http://127.0.0.1:9090
```

---

## 模块 2：后端 API、认证与权限控制

### 1. 前置语法补充

- Go Web：函数签名 `func Xxx(c *gin.Context)`，`c.JSON()` 返回响应，`c.ShouldBindJSON()` 绑定请求体。
- Gin 路由：`r.GET`、`r.POST`、`Group()`、中间件 `Use()`。
- 闭包：`func(c *gin.Context) { ... }` 作为 handler 或 middleware。
- JWT：登录后生成 token；后续请求通过 `Authorization: Bearer <token>` 携带。
- Go map：`map[string]bool` 用于角色白名单。

### 2. 核心源码突破

| 文件 | 核心函数 / 对象 | 作用 | 优先级 |
|---|---|---|---|
| `application/backend/router/router.go` | `SetupRouter()` | 全部 HTTP API 的总索引；定义公开登录注册接口、鉴权接口、角色权限接口、静态文件托管 | 最高 |
| `application/backend/middleware/auth.go` | `Auth()` | 解析 JWT，把 `userID / username / role` 放进 Gin Context | 最高 |
| `application/backend/middleware/auth.go` | `Roles(roles ...string)` | 按角色拦截接口，如“原料供应商”“榨油厂”“运输人员”等 | 最高 |
| `application/backend/controller/user.go` | `Register()` | 用户注册，默认 `pending`，等待管理员审核 | 高 |
| `application/backend/controller/user.go` | `Login()` | 校验密码、账号状态，签发 JWT | 高 |
| `application/backend/controller/user.go` | `GetInfo()` | 前端刷新页面后恢复用户身份与角色 | 中 |
| `application/backend/controller/user.go` | `ListUsers()` / `UpdateUserStatus()` | 管理员审核、禁用、拒绝账号 | 中 |
| `application/backend/pkg/core.go` | `HashPassword()` | SHA-256 密码哈希 | 中 |
| `application/backend/pkg/core.go` | `GenToken()` / `ParseToken()` | JWT 生成与解析 | 高 |
| `application/backend/pkg/core.go` | `Log()` | 写操作日志 | 中 |

### 阅读提示

先把接口分成两类：

```text
无需登录：
  POST /api/register
  POST /api/login

需要登录：
  /api/* + middleware.Auth()

需要指定角色：
  /api/batches, /api/factory/decision, /api/transport/start 等 + middleware.Roles(...)
```

不要先纠结每个 HTTP 状态码。核心是理解：

```text
JWT -> Auth 中间件解析 -> Gin Context 保存用户信息 -> Roles 中间件判断角色 -> Controller 执行业务
```

---

## 模块 3：核心业务工作流、MySQL 数据模型与本地存证

这是最重要模块。项目是否理解，主要看你是否能讲清楚“一个批次如何从原料供应商走到零售商并被追溯”。

### 1. 前置语法补充

- Go `database/sql`：`DB.Exec()`、`DB.Query()`、`QueryRow().Scan()`、事务 `Begin / Commit / Rollback`。
- SQL：`CREATE TABLE`、`INSERT`、`UPDATE`、`JOIN`、`LEFT JOIN`、`WHERE`、`ORDER BY`。
- JSON 字段：MySQL `JSON` 字段，Go 里用 `map[string]interface{}`、`json.Marshal()`、`json.RawMessage`。
- Go struct 临时请求体：`var req struct { ... }`。
- 状态机思维：不要逐字段背诵，重点看 `status` 如何变化。

### 2. 核心源码突破

| 文件 | 核心函数 / 对象 | 作用 | 阅读目标 |
|---|---|---|---|
| `application/backend/pkg/core.go` | `InitDB()` | 创建数据库、创建表、写入种子数据 | 明白系统核心表结构 |
| `application/backend/pkg/core.go` | `schema` | 定义 `users / batches / transport_tasks / transport_nodes / evidence_records / logs` 等表 | 先看表，不看细节字段 |
| `application/backend/pkg/core.go` | `seedData()` / `seedBatch()` | 初始化演示账号和演示批次 | 理解系统预置数据如何形成完整链路 |
| `application/backend/pkg/core.go` | `CreateEvidence()` / `CreateEvidenceAt()` | 生成 `data_hash / previous_hash / transaction_hash / block_hash`，写入 `evidence_records` | 理解当前主业务的“存证”实现 |
| `application/backend/pkg/core.go` | `GenerateTraceCode()` | 生成溯源码 | 明白批次唯一 ID 来源 |
| `application/backend/pkg/core.go` | `DomesticRoute()` / `SampleRoute()` | 生成模拟运输路线和 GPS 节点 | 明白运输轨迹数据怎么产生 |
| `application/backend/controller/business.go` | `CreateBatch()` | 原料供应商创建原料批次草稿 | 业务入口 |
| `application/backend/controller/business.go` | `SubmitBatch()` | 提交原料批次，并写“原料批次存证” | 第一次存证 |
| `application/backend/controller/business.go` | `FactoryDecision()` | 榨油厂接收或拒收原料 | 状态分支关键点 |
| `application/backend/controller/business.go` | `SubmitProcessing()` | 榨油厂提交加工生产信息 | 加工阶段 |
| `application/backend/controller/business.go` | `CreateTransport()` | 榨油厂创建运输任务，事务内同步更新批次状态 | 事务关键点 |
| `application/backend/controller/business.go` | `TransportDecision()` | 运输人员接收或退回运输任务 | 运输状态入口 |
| `application/backend/controller/business.go` | `StartTransport()` | 开始运输 | 状态进入 `in_transit` |
| `application/backend/controller/business.go` | `GenerateNodes()` | 生成 GPS / 温湿度节点，并写运输过程存证 | 运输数据核心 |
| `application/backend/controller/business.go` | `CompleteTransport()` | 运输完成，等待零售商确认 | 运输收尾 |
| `application/backend/controller/business.go` | `RetailDecision()` | 零售商确认收货或退回 | 主流程闭环 |
| `application/backend/controller/business.go` | `TraceDetail()` | 根据溯源码聚合批次、运输、节点、存证、更正、退回记录 | 追溯查询入口 |
| `application/backend/controller/business.go` | `visibleBatches()` / `batchVisibilityClause()` / `canAccessBatch()` | 根据角色过滤可见数据 | 权限与数据查询边界 |

### 核心状态流

优先记这条主路径：

```text
raw_draft
  -> pending_factory
  -> factory_received
  -> processed
  -> pending_transport
  -> transport_accepted
  -> in_transit
  -> pending_retail
  -> completed
```

对应业务角色：

```text
原料供应商
  创建草稿 / 修改草稿 / 提交批次
榨油厂
  接收原料 / 拒收原料 / 提交加工信息 / 创建运输任务
运输人员
  接收运输任务 / 开始运输 / 生成 GPS 与温湿度节点 / 完成运输
零售商
  确认收货 / 退回运输环节
监管机构
  查看全量追溯和存证
系统管理员
  用户审核与日志查看
```

### 当前存证机制要点

当前后端没有直接调用 Fabric SDK，而是在 MySQL 表 `evidence_records` 中记录存证信息：

```text
业务动作
  -> Controller 更新业务表
  -> 调用 pkg.CreateEvidence(...)
  -> CreateEvidenceAt 计算 data_hash
  -> 查询上一条 previous_hash
  -> 生成 transaction_hash / block_hash
  -> 插入 evidence_records
```

这意味着源码阅读时应先把 `CreateEvidenceAt()` 当成“本项目当前运行版的简化链上存证层”。Fabric 链码后面再独立看。

---

## 模块 4：前端 Vue 管理台、页面路由与接口封装

### 1. 前置语法补充

- JavaScript ES6：`import / export`、箭头函数、Promise、`async / await`、模板字符串。
- Vue2：单文件组件 `.vue`、`template / script / style`、`data()`、`computed`、`created()`、`methods`。
- Vue Router：路由表、`meta.roles`、路由守卫。
- Vuex：`state / mutations / actions`，理解为前端全局状态。
- Axios：请求拦截器、响应拦截器、统一 baseURL。
- Element UI：只需知道表格、表单、按钮、弹窗组件的用途，不必深挖组件内部。

### 2. 核心源码突破

| 文件 | 核心对象 / 函数 | 作用 | 阅读目标 |
|---|---|---|---|
| `application/web/package.json` | `scripts`、`dependencies` | Vue2、Vue Router、Vuex、Axios、Element UI、高德地图依赖 | 知道前端技术栈 |
| `application/web/src/main.js` | `new Vue({ router, store })` | 前端应用入口，挂载 Router 和 Vuex | 明白前端启动入口 |
| `application/web/src/router/index.js` | `constantRoutes` | 页面路由与角色权限声明 | 对照后端角色读页面 |
| `application/web/src/permission.js` | `router.beforeEach` | 路由守卫：无 token 跳登录；无角色则拉取用户信息；角色不匹配跳无权限页 | 明白前端权限入口 |
| `application/web/src/store/modules/user.js` | `login()` / `getInfo()` / `logout()` | 保存 token、用户角色、组织信息 | 明白前端如何保存登录态 |
| `application/web/src/utils/request.js` | Axios 实例、拦截器 | 自动加 `Authorization`，统一处理响应和错误 | 前后端通信关键 |
| `application/web/src/api/user.js` | 用户接口封装 | 登录、注册、管理员用户管理 | 与后端 `user.go` 对照 |
| `application/web/src/api/business.js` | 业务接口封装 | 批次、加工、运输、零售、追溯、存证接口 | 与后端 `business.go` 对照 |
| `application/web/src/views/batches/index.vue` | 原材料页面 | 供应商创建、编辑、提交批次 | 看业务入口 |
| `application/web/src/views/pressing/index.vue` | 压榨加工页面 | 榨油厂接收原料、提交加工、创建运输任务 | 看加工阶段 |
| `application/web/src/views/transport/index.vue` | 运输页面 | 接收任务、启运、生成节点、完成运输 | 看运输阶段 |
| `application/web/src/views/retail/index.vue` | 零售页面 | 确认收货或退回 | 看闭环阶段 |
| `application/web/src/views/trace/index.vue` | 全流程追溯页面 | 聚合展示原料、加工、运输、零售、存证时间线和地图轨迹 | 看系统价值展示 |
| `application/web/src/views/evidence/index.vue` | 存证查询页面 | 展示 evidence 记录、哈希、交易哈希、区块哈希 | 看“可信记录”展示 |

### 阅读提示

前端不要从 UI 样式读起，按这条线读：

```text
页面按钮点击
  -> 调用 src/api/business.js
  -> src/utils/request.js 自动带 token
  -> 后端 router.go 对应接口
  -> controller/business.go 执行业务
  -> 返回 JSON
  -> 页面刷新表格 / 时间线 / 地图
```

---

## 模块 5：Fabric 链码与区块链网络脚本

### 1. 前置语法补充

- Go 方法接收者：`func (s *SmartContract) Xxx(...)` 类似 C++ 成员函数。
- Fabric Chaincode 基础：`contractapi.Contract`、`TransactionContextInterface`、`ctx.GetStub()`。
- World State：`PutState(key, value)` 写状态，`GetState(key)` 读状态，`GetStateByRange` 遍历，`GetHistoryForKey` 查历史。
- JSON 序列化：链码把结构体转 JSON 字节写入账本。
- Bash：函数、环境变量、`case` 参数解析、Docker Compose 启停。

### 2. 核心源码突破

| 文件 | 核心对象 / 函数 | 作用 | 阅读目标 |
|---|---|---|---|
| `blockchain/chaincode/trace.go` | `main()` | 创建并启动 Fabric chaincode，注册 `SmartContract` | 链码入口 |
| `blockchain/chaincode/chaincode/model.go` | `User`、`Fruit`、`Farmer_input`、`Factory_input`、`Driver_input`、`Shop_input`、`HistoryQueryResult` | 链码账本中的数据模型 | 先看结构体字段，不纠结命名历史 |
| `blockchain/chaincode/chaincode/smartcontract.go` | `SmartContract` | Fabric 合约结构体 | 合约方法的宿主 |
| `blockchain/chaincode/chaincode/smartcontract.go` | `RegisterUser()` | 用户注册上链 | 账本用户入口 |
| `blockchain/chaincode/chaincode/smartcontract.go` | `Uplink()` | 根据用户类型写入不同环节信息，并返回交易 ID | 链码核心写入函数 |
| `blockchain/chaincode/chaincode/smartcontract.go` | `AddFruit()` | 把产品加入用户产品列表 | 用户-产品关联 |
| `blockchain/chaincode/chaincode/smartcontract.go` | `GetUserType()` / `GetUserInfo()` | 查询用户信息 | 辅助读取 |
| `blockchain/chaincode/chaincode/smartcontract.go` | `GetFruitInfo()` / `GetFruitList()` / `GetAllFruitInfo()` | 查询产品、用户产品列表、全量产品 | 查询接口 |
| `blockchain/chaincode/chaincode/smartcontract.go` | `GetFruitHistory()` | 查询某个溯源码的历史交易记录 | 区块链不可篡改历史的关键接口 |
| `blockchain/network/network.sh` | `up / createChannel / deployCC / invoke / query` 分支 | 启停 Fabric 测试网络、建通道、部署和调用链码 | 理解链码部署路径 |

### 阅读提示

这部分不要和当前 Go 后端业务混在一起。推荐单独理解：

```text
Fabric 网络启动
  -> 创建组织证书和节点
  -> 创建 channel
  -> 部署 chaincode
  -> 调用 RegisterUser / Uplink
  -> World State 保存 User / Fruit
  -> GetFruitHistory 查询历史
```

它代表项目的区块链扩展方向；而当前本地业务系统可以先按 MySQL + 哈希链存证理解。

---

# 第二阶段：模块交互与数据流向

## 1. 初始化流程

### A. 本地应用初始化流程

```text
start-local.ps1
  1. 检查 Docker / Go / npm
  2. 拉取并启动 mysql:8.0
  3. 等待 MySQL healthcheck 通过
  4. 进入 application/web，执行 npm install / npm run build:prod
  5. 将前端 dist 复制到 application/backend/dist
  6. 进入 application/backend，go build 生成后端可执行文件
  7. 启动 oil-supervision-server.exe
  8. 打开 http://127.0.0.1:9090
```

### B. Go 后端启动流程

```text
application/backend/main.go: main()
  -> settings.Init()
       读取 settings/config.yaml
  -> pkg.InitDB()
       连接 MySQL
       CREATE DATABASE IF NOT EXISTS
       创建 users / batches / transport_tasks / evidence_records 等表
       seedData() 写入演示用户和演示批次
  -> router.SetupRouter()
       创建 Gin Engine
       配置 CORS
       托管前端静态文件 dist
       注册公开接口 /api/login /api/register
       注册需要 Auth 和 Roles 的业务接口
  -> Run(:9090)
       HTTP 服务开始接收请求
```

### C. 前端初始化流程

```text
浏览器打开 /
  -> 后端返回 dist/index.html
  -> application/web/src/main.js 创建 Vue 实例
  -> 挂载 router 和 store
  -> 加载 permission.js 路由守卫
  -> 如果没有 token：跳 /login
  -> 如果有 token 但没有角色：调用 user/getInfo
  -> 根据 router/index.js 中 meta.roles 判断页面权限
```

### D. Fabric 网络初始化流程

```text
blockchain/network/network.sh up
  -> 检查 Fabric 工具和 Docker 镜像
  -> 生成组织证书材料
  -> 启动 peer / orderer / CA 等容器

network.sh createChannel
  -> 创建并加入 channel

network.sh deployCC
  -> 调用 scripts/deployCC.sh
  -> 打包、安装、批准、提交 chaincode
```

注意：D 是 Fabric 侧初始化，不是当前 Web 应用启动的必经路径。

---

## 2. 核心数据流（Data Flow）

## 场景一：用户登录

```text
前端 login 页面
  -> store/modules/user.js: login(form)
  -> api/user.js: POST /login
  -> utils/request.js: axios 发送请求
  -> router.go: POST /api/login -> controller.Login
  -> user.go: Login()
       查询 users 表
       比对 HashPassword(password)
       检查 status 是否 approved
       pkg.GenToken 生成 JWT
       写 login_logs
  -> 返回 jwt + user
  -> 前端保存 token 到 Cookie / Vuex
  -> 后续请求自动带 Authorization: Bearer <token>
```

## 场景二：原料供应商提交批次

```text
前端 原材料管理页面
  -> api/business.js: createBatch / submitBatch
  -> utils/request.js 自动带 JWT
  -> router.go
       POST /api/batches
       POST /api/batches/submit
       中间件：Auth + Roles("原料供应商")
  -> business.go
       CreateBatch(): 插入 batches，status = raw_draft
       SubmitBatch(): status -> pending_factory
       pkg.CreateEvidence(): 写“原料批次存证”
  -> pkg/core.go
       CreateEvidenceAt(): 计算 data_hash / previous_hash / transaction_hash / block_hash
       插入 evidence_records
  -> 前端刷新批次列表
```

## 场景三：榨油厂接收并加工

```text
榨油厂页面
  -> GET /api/batches
       visibleBatches(includeFactoryPool=true)
       能看到待接收的 pending_factory 批次

点击“接收”
  -> POST /api/factory/decision
  -> FactoryDecision(Accept=true)
       batches.status -> factory_received
       batches.oil_factory_id = 当前榨油厂
       CreateEvidence("原料接收存证")

提交加工信息
  -> POST /api/factory/processing
  -> SubmitProcessing()
       processing_data 写入 JSON
       batches.status -> processed
       CreateEvidence("加工生产存证")
```

## 场景四：创建并执行运输任务

```text
榨油厂创建运输任务
  -> POST /api/transport
  -> CreateTransport()
       开启 SQL 事务
       插入 transport_tasks，status = pending_accept
       更新 batches.status = pending_transport
       设置 transporter_id / retailer_id
       提交事务
       CreateEvidence("运输任务存证")

运输人员接收任务
  -> POST /api/transport/decision
  -> TransportDecision(Accept=true)
       transport_tasks.status -> accepted
       batches.status -> transport_accepted
       CreateEvidence("运输接收存证")

运输人员开始运输
  -> POST /api/transport/start
  -> StartTransport()
       transport_tasks.status -> in_transit
       batches.status -> in_transit
       CreateEvidence("运输启运存证")

生成 GPS 与温湿度节点
  -> POST /api/transport/nodes
  -> GenerateNodes()
       DomesticRoute() 生成路线
       SampleRoute() 采样节点
       插入 transport_nodes
       CreateEvidence("运输过程存证")

完成运输
  -> POST /api/transport/complete
  -> CompleteTransport()
       transport_tasks.status -> pending_retail
       batches.status -> pending_retail
       CreateEvidence("运输完成存证")
```

## 场景五：零售商确认收货并形成完整追溯

```text
零售商页面
  -> POST /api/retail/decision
  -> RetailDecision(Accept=true)
       receipt_data 写入 JSON
       batches.status -> completed
       transport_tasks.status -> completed
       CreateEvidence("零售收货存证")

追溯页面
  -> GET /api/trace-batches
       获取当前角色可看的批次
  -> GET /api/trace/:code
  -> TraceDetail()
       batchByID()
       evidenceByBatch()
       nodesByBatch()
       correctionsByBatch()
       rejectionsByBatch()
       transportByBatch()
  -> 返回聚合数据
  -> 前端 trace/index.vue 展示：
       原料信息
       加工信息
       运输任务
       GPS / 温湿度节点
       零售收货
       存证时间线
       交易哈希 / 区块哈希
```

---

## 3. 通信机制

## A. 前后端通信：HTTP + JSON + JWT

```text
Vue 页面
  -> src/api/*.js
  -> Axios
  -> HTTP /api/xxx
  -> Gin Router
  -> Controller
  -> JSON response
```

关键点：

- 请求体基本是 JSON。
- 登录后前端保存 JWT。
- `utils/request.js` 在请求头里自动加 `Authorization`。
- 后端 `middleware.Auth()` 解析 JWT。
- 后端通过 Gin Context 传递 `userID / username / role`。

## B. 后端内部通信：函数调用 + Gin Context + 全局 DB

```text
router.go
  -> controller/*.go
  -> pkg.DB
  -> MySQL

controller/*.go
  -> pkg.CreateEvidence()
  -> pkg.Log()
  -> pkg.GenerateTraceCode()
  -> pkg.DomesticRoute() / pkg.SampleRoute()
```

关键点：

- 没有消息队列。
- 没有事件总线。
- 没有复杂回调链。
- 没有依赖注入框架。
- 数据库连接是 `pkg.DB` 这个全局变量。
- 请求级用户状态放在 `gin.Context`。

## C. 权限机制：路由中间件 + 数据过滤

```text
接口级权限：
  router.go 中 middleware.Roles(...)

数据级权限：
  business.go 中 batchVisibilityClause()
  business.go 中 canAccessBatch()
```

理解这两层即可：

- `Roles()` 决定“这个角色能不能访问接口”。
- `batchVisibilityClause()` 决定“这个角色能看到哪些批次”。

## D. 存证机制：业务动作同步写入 evidence_records

```text
Controller 更新业务数据
  -> 同步调用 pkg.CreateEvidence
  -> evidence_records 追加一条记录
  -> previous_hash 指向同一批次上一条 block_hash
  -> 形成批次维度的哈希链
```

这是一种同步写入，不是异步队列，也不是回调触发。

## E. Fabric 链码通信机制：链码通过 Fabric Stub 读写账本

```text
Fabric Client 调用链码方法
  -> SmartContract.RegisterUser / Uplink / GetFruitInfo / GetFruitHistory
  -> ctx.GetStub().PutState / GetState / GetHistoryForKey
  -> World State 与交易历史
```

当前仓库里 Fabric 链码和网络脚本存在，但后端应用代码中没有看到直接调用 Fabric Gateway / SDK 的请求链路。因此阅读时应把它作为“区块链账本实现模块”单独理解，不要误认为每个后端业务接口都会真实调用 Fabric 链码。

---

# 三、建议学习节奏

## 第 1 天：跑通系统 + 读入口

目标：能解释项目怎么启动。

读：

- `start-local.ps1`
- `docker-compose.yml`
- `application/backend/main.go`
- `application/backend/settings/config.yaml`
- `application/backend/router/router.go`

产出：画出“本地启动流程图”。

## 第 2 天：读认证、角色和路由

目标：能解释“为什么某个角色能看到某些页面和接口”。

读：

- `application/backend/middleware/auth.go`
- `application/backend/controller/user.go`
- `application/web/src/permission.js`
- `application/web/src/router/index.js`
- `application/web/src/store/modules/user.js`

产出：列出 6 个角色各自能访问的主要页面和接口。

## 第 3-4 天：读核心业务状态机

目标：能从 `raw_draft` 讲到 `completed`。

读：

- `application/backend/controller/business.go`
- `application/backend/pkg/core.go` 中 `schema / CreateEvidenceAt / DomesticRoute / SampleRoute`
- `application/web/src/api/business.js`

产出：画出批次状态流转图，并标注每个状态由哪个 Controller 函数改变。

## 第 5 天：读前端页面如何驱动业务

目标：能从一个按钮点击追到后端 Controller。

读：

- `application/web/src/utils/request.js`
- `application/web/src/views/batches/index.vue`
- `application/web/src/views/pressing/index.vue`
- `application/web/src/views/transport/index.vue`
- `application/web/src/views/retail/index.vue`
- `application/web/src/views/trace/index.vue`
- `application/web/src/views/evidence/index.vue`

产出：任选一个动作，例如“运输人员生成节点”，从页面按钮追到 SQL 写入。

## 第 6 天：读 Fabric 链码

目标：能解释链码如何保存和查询溯源数据。

读：

- `blockchain/chaincode/trace.go`
- `blockchain/chaincode/chaincode/model.go`
- `blockchain/chaincode/chaincode/smartcontract.go`
- `blockchain/network/network.sh`

产出：说明 `Uplink()` 如何根据用户类型写入不同环节数据，以及 `GetFruitHistory()` 为什么能查历史。

---

# 四、源码阅读时可以暂时跳过的部分

为了保持效率，以下内容第一轮可以跳过或只扫一眼：

- `application/backend/distbak/`、`application/backend/dist/`：构建产物或备份产物，不是第一轮重点。
- 大量 CSS / SCSS：只影响界面样式。
- `application/web/src/icons/`：图标资源。
- `blockchain/network/organizations/`、`channel-artifacts/`、各类生成证书目录：Fabric 运行生成物或配置细节，第一轮不必深挖。
- Fabric 网络脚本中的所有兼容分支：先理解 `up / createChannel / deployCC / down` 四类命令即可。

---

# 五、最终验收标准

完成第一轮源码阅读后，你应能不用看代码回答这些问题：

1. 本地系统启动时，MySQL、前端、后端分别如何启动？
2. 用户登录后，JWT 如何在前端保存，又如何在后端解析？
3. 一个原料批次从创建到完成收货，中间经过哪些状态？
4. 每个状态由哪个角色、哪个 API、哪个 Controller 函数推动？
5. 运输轨迹和温湿度节点从哪里来，存在什么表里？
6. 追溯详情页的数据是一次性从哪里聚合出来的？
7. 当前主业务“存证”写在哪里，哈希链如何形成？
8. Fabric 链码的 `Uplink()` 和后端 `CreateEvidenceAt()` 分别解决什么问题？
9. 当前项目哪些部分是运行主链路，哪些部分更像区块链扩展或实验模块？

只要能回答这些问题，就说明你已经理解了项目主干，可以再按需求深入局部代码。
