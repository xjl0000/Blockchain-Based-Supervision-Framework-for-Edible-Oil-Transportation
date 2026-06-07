<template>
  <div class="page">
    <div class="page-head">
      <div>
        <h2>运输管理</h2>
        <p>运输人员接收任务、确认启运，并上传GPS定位、温度与湿度记录。</p>
      </div>
    </div>

    <el-table :data="list" border>
      <el-table-column prop="trace_code" label="溯源码" width="170" />
      <el-table-column prop="product_name" label="产品" />
      <el-table-column prop="vehicle_no" label="运输车辆" />
      <el-table-column label="运输路线" min-width="180">
        <template slot-scope="{row}">{{ row.start_city }} → {{ row.end_city }}</template>
      </el-table-column>
      <el-table-column prop="transporter" label="运输人员" />
      <el-table-column prop="retailer" label="目标零售商" />
      <el-table-column label="状态" width="130">
        <template slot-scope="{row}"><el-tag :type="tagType(row.status)">{{ statusName(row.status) }}</el-tag></template>
      </el-table-column>
      <el-table-column label="操作" min-width="280">
        <template slot-scope="{row}">
          <el-button size="mini" @click="showMap(row)">轨迹详情</el-button>
          <template>
            <el-button v-if="row.status==='pending_accept'" size="mini" type="success" @click="driverDecision(row,true)">接收</el-button>
            <el-button v-if="row.status==='pending_accept'" size="mini" type="danger" @click="driverDecision(row,false)">退回</el-button>
            <el-button v-if="row.status==='accepted'" size="mini" type="primary" @click="start(row)">开始运输</el-button>
            <el-button v-if="row.status==='in_transit'" size="mini" type="warning" @click="generate(row)">更新定位数据</el-button>
            <el-button v-if="row.status==='in_transit'" size="mini" type="success" @click="complete(row)">完成运输</el-button>
          </template>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog title="发起食用油运输任务" :visible.sync="taskVisible" width="720px">
      <el-form label-width="110px">
        <el-form-item label="加工批次">
          <el-select v-model="task.batchID">
            <el-option v-for="b in available" :key="b.id" :label="b.trace_code+' / '+b.material_name" :value="b.id" />
          </el-select>
        </el-form-item>
        <el-row :gutter="14">
          <el-col :span="12">
            <el-form-item label="运输人员">
              <el-select v-model="task.transporterID">
                <el-option v-for="option in drivers" :key="option.id" :label="option.name+' / '+option.organization" :value="option.id" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="目标零售商">
              <el-select v-model="task.retailerID">
                <el-option v-for="option in retailers" :key="option.id" :label="option.name+' / '+option.organization" :value="option.id" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="14">
          <el-col :span="12"><el-form-item label="司机姓名"><el-input v-model="task.driverName" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="车辆牌照"><el-input v-model="task.vehicleNo" /></el-form-item></el-col>
        </el-row>
        <el-row :gutter="14">
          <el-col :span="12"><el-form-item label="产品名称"><el-input v-model="task.productName" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="产品数量"><el-input-number v-model="task.productQuantity" /></el-form-item></el-col>
        </el-row>
        <el-row :gutter="14">
          <el-col :span="12">
            <el-form-item label="起点城市">
              <el-select v-model="startKey" @change="setStart"><el-option v-for="(v,k) in cities" :key="k" :label="k" :value="k" /></el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="目的城市">
              <el-select v-model="endKey" @change="setEnd"><el-option v-for="(v,k) in cities" :key="k" :label="k" :value="k" /></el-select>
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="运输说明"><el-input v-model="task.note" type="textarea" /></el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="taskVisible=false">取消</el-button>
        <el-button type="primary" @click="saveTask">提交并生成存证</el-button>
      </span>
    </el-dialog>

    <el-dialog
      title="运输轨迹与温湿度记录"
      :visible.sync="mapVisible"
      width="960px"
      @opened="renderMap"
      @closed="destroyMap"
    >
      <div class="route-summary">
        <span><b>溯源码：</b>{{ selectedTask && selectedTask.trace_code }}</span>
        <span><b>运输路线：</b>{{ selectedTask && selectedTask.start_city }} → {{ selectedTask && selectedTask.end_city }}</span>
        <span><b>GPS节点：</b>{{ mapNodes.length }} 个</span>
      </div>
      <div v-loading="mapLoading" class="map-board">
        <div ref="amapContainer" class="amap-container" />
        <div v-if="mapError" class="map-error">
          <i class="el-icon-warning-outline" />
          <span>{{ mapError }}</span>
        </div>
        <div class="legend">
          <span><i class="dot start-dot" />运输起点</span>
          <span><i class="dot node-dot" />GPS定位节点</span>
          <span><i class="dot end-dot" />运输终点</span>
        </div>
      </div>
      <el-table :data="mapNodes" height="260" border>
        <el-table-column prop="seq" label="节点" width="60" />
        <el-table-column prop="longitude" label="经度" />
        <el-table-column prop="latitude" label="纬度" />
        <el-table-column prop="temperature" label="温度℃" />
        <el-table-column prop="humidity" label="湿度%" />
        <el-table-column prop="recorded_at" label="上传时间" width="180" />
      </el-table>
    </el-dialog>
  </div>
</template>

<script>
import { mapGetters } from 'vuex'
import { transports, batches, roleOptions, createTransport, transportDecision, startTransport, generateNodes, completeTransport, retailDecision, nodes } from '@/api/business'
import { statusName, tagType } from '@/utils/workflow'
import { loadAMap } from '@/utils/amap'

const cities = {
  '哈尔滨市': [126.642464, 45.756967, '黑龙江省'],
  '北京市': [116.407526, 39.90403, '北京市'],
  '上海市': [121.473701, 31.230416, '上海市'],
  '青岛市': [120.38264, 36.067082, '山东省'],
  '郑州市': [113.625368, 34.7466, '河南省'],
  '成都市': [104.066541, 30.572269, '四川省'],
  '广州市': [113.264385, 23.129112, '广东省'],
  '西安市': [108.93977, 34.341574, '陕西省']
}

export default {
  data: () => ({
    timer: null,
    map: null,
    mapLoading: false,
    mapError: '',
    list: [],
    available: [],
    drivers: [],
    retailers: [],
    taskVisible: false,
    mapVisible: false,
    mapNodes: [],
    selectedTask: null,
    startKey: '哈尔滨市',
    endKey: '北京市',
    cities,
    task: {
      driverName: '王志强',
      vehicleNo: '黑A·E6608',
      productName: '一级压榨食用油',
      productQuantity: 18.6,
      note: '食品级专用罐车运输，全程铅封并记录定位及温湿度'
    }
  }),
  computed: {
    ...mapGetters(['role'])
  },
  created() {
    this.load()
    this.timer = setInterval(this.autoUpdate, 30000)
  },
  beforeDestroy() {
    clearInterval(this.timer)
    this.destroyMap()
  },
  methods: {
    statusName,
    tagType,
    load() {
      transports().then(r => { this.list = r.data }).catch(() => {})
      batches().then(r => { this.available = r.data.filter(item => item.status === 'processed') })
      roleOptions('运输人员').then(r => { this.drivers = r.data })
      roleOptions('零售商').then(r => { this.retailers = r.data })
    },
    autoUpdate() {
      if (this.role !== '运输人员') return
      const active = this.list.filter(item => item.status === 'in_transit')
      Promise.all(active.map(item => generateNodes(item.id, 1))).then(() => {
        if (active.length) this.load()
      }).catch(() => {})
    },
    openTask() {
      this.taskVisible = true
      this.setStart(this.startKey)
      this.setEnd(this.endKey)
    },
    setStart(k) {
      const p = cities[k]
      Object.assign(this.task, { startCity: k, startProvince: p[2], startLng: p[0], startLat: p[1] })
    },
    setEnd(k) {
      const p = cities[k]
      Object.assign(this.task, { endCity: k, endProvince: p[2], endLng: p[0], endLat: p[1] })
    },
    saveTask() {
      createTransport(this.task).then(r => {
        this.$message.success(r.message)
        this.taskVisible = false
        this.load()
      })
    },
    driverDecision(row, accept) {
      this.$prompt(accept ? '填写接收备注' : '填写退回原因', '运输任务确认', { inputValue: accept ? '车辆及任务信息核验通过' : '' })
        .then(x => transportDecision({ id: row.id, accept, reason: x.value }))
        .then(x => { this.$message.success(x.message); this.load() })
    },
    start(row) {
      this.$confirm('确认车辆已启运？').then(() => startTransport(row.id)).then(x => { this.$message.success(x.message); this.load() })
    },
    generate(row) {
      generateNodes(row.id, 8).then(x => { this.$message.success(x.message); this.showMap(row) })
    },
    complete(row) {
      this.$confirm('确认运输已完成并提交零售商收货？').then(() => completeTransport(row.id)).then(x => { this.$message.success(x.message); this.load() })
    },
    retail(row, accept) {
      this.$prompt(accept ? '填写收货核验结果' : '填写拒收退回原因', '零售收货确认', { inputValue: accept ? '数量一致、包装完整、质量抽检合格' : '' })
        .then(x => retailDecision({ batchID: row.batch_id, accept, reason: x.value, data: { result: '确认收货', quality: x.value, quantity: row.product_quantity }}))
        .then(x => { this.$message.success(x.message); this.load() })
    },
    showMap(row) {
      this.selectedTask = row
      nodes(row.id).then(x => {
        this.mapNodes = x.data
        this.mapVisible = true
      })
    },
    async renderMap() {
      if (!this.selectedTask || !this.$refs.amapContainer) return
      this.destroyMap()
      this.mapLoading = true
      this.mapError = ''
      try {
        const AMap = await loadAMap()
        const start = [Number(this.selectedTask.start_lng), Number(this.selectedTask.start_lat)]
        const end = [Number(this.selectedTask.end_lng), Number(this.selectedTask.end_lat)]
        const nodePath = this.mapNodes.map(node => [Number(node.longitude), Number(node.latitude)])
        const path = [start, ...nodePath, end]
        this.map = new AMap.Map(this.$refs.amapContainer, {
          viewMode: '2D',
          zoom: 5,
          center: start,
          mapStyle: 'amap://styles/whitesmoke',
          resizeEnable: true
        })
        this.map.addControl(new AMap.Scale())
        this.map.addControl(new AMap.ToolBar({ position: 'RT' }))

        const overlays = []
        const polyline = new AMap.Polyline({
          path,
          strokeColor: '#409EFF',
          strokeWeight: 6,
          strokeOpacity: 0.9,
          lineJoin: 'round',
          lineCap: 'round',
          showDir: true
        })
        overlays.push(polyline)

        const startMarker = this.createEndpointMarker(AMap, start, '起', '#67C23A', this.selectedTask.start_city)
        const endMarker = this.createEndpointMarker(AMap, end, '终', '#F56C6C', this.selectedTask.end_city)
        overlays.push(startMarker, endMarker)

        this.mapNodes.forEach(node => {
          const marker = new AMap.CircleMarker({
            center: [Number(node.longitude), Number(node.latitude)],
            radius: 7,
            strokeColor: '#FFFFFF',
            strokeWeight: 2,
            fillColor: '#409EFF',
            fillOpacity: 1,
            zIndex: 20
          })
          marker.on('click', () => this.openNodeInfo(AMap, node))
          overlays.push(marker)
        })

        this.map.add(overlays)
        this.map.setFitView(overlays, false, [55, 55, 55, 55], 11)
      } catch (error) {
        this.mapError = `地图加载失败：${error.message || '请检查高德地图 Key 配置及网络连接'}`
      } finally {
        this.mapLoading = false
      }
    },
    createEndpointMarker(AMap, position, label, color, title) {
      return new AMap.Marker({
        position,
        anchor: 'center',
        title,
        content: `<div style="width:32px;height:32px;border-radius:50%;background:${color};color:#fff;border:3px solid #fff;box-shadow:0 2px 8px rgba(0,0,0,.28);display:flex;align-items:center;justify-content:center;font-weight:700;">${label}</div>`,
        zIndex: 30
      })
    },
    openNodeInfo(AMap, node) {
      const content = [
        '<div style="min-width:190px;line-height:1.8;">',
        `<strong>运输定位节点 ${node.seq}</strong>`,
        `<div>温度：${node.temperature} ℃</div>`,
        `<div>湿度：${node.humidity} %</div>`,
        `<div>上传时间：${node.recorded_at}</div>`,
        '</div>'
      ].join('')
      new AMap.InfoWindow({ content, offset: new AMap.Pixel(0, -10) })
        .open(this.map, [Number(node.longitude), Number(node.latitude)])
    },
    destroyMap() {
      if (this.map) {
        this.map.destroy()
        this.map = null
      }
    }
  }
}
</script>

<style scoped>
.route-summary {
  display: flex;
  gap: 28px;
  padding: 0 2px 12px;
  color: #536471;
}
.map-board {
  position: relative;
  overflow: hidden;
  min-height: 430px;
  margin-bottom: 14px;
  border: 1px solid #d5e4e8;
  border-radius: 8px;
  background: #f3f7f8;
}
.amap-container {
  width: 100%;
  height: 390px;
}
.map-error {
  position: absolute;
  inset: 0 0 39px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  color: #909399;
  background: #f3f7f8;
}
.map-error i {
  font-size: 32px;
}
.legend {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 30px;
  height: 39px;
  color: #60717e;
  background: #fff;
}
.legend span {
  display: inline-flex;
  align-items: center;
}
.dot {
  width: 10px;
  height: 10px;
  margin-right: 7px;
  border-radius: 50%;
}
.start-dot { background: #67c23a; }
.node-dot { background: #409eff; }
.end-dot { background: #f56c6c; }
</style>
