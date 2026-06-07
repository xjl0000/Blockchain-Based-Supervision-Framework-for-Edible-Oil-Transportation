<template>
  <div class="page trace-page">
    <div class="page-head">
      <div><h2>{{ detail ? '完整追溯信息' : '产品全流程追溯' }}</h2><p>{{ detail ? '依次核验原料、加工、运输、零售和各环节可信时间线。' : '选择有权限查看的批次，进入完整追溯信息。' }}</p></div>
      <el-button v-if="detail" icon="el-icon-back" @click="backToList">返回溯源码列表</el-button>
    </div>
    <div v-if="!detail" class="panel batch-list">
      <div class="toolbar"><el-input v-model="keyword" clearable placeholder="搜索溯源码、原料或产地" prefix-icon="el-icon-search" style="width:320px" /><span>共 {{ filtered.length }} 个可追溯批次</span></div>
      <el-table :data="filtered" border highlight-current-row @row-click="open">
        <el-table-column prop="trace_code" label="溯源码" width="180" /><el-table-column prop="material_name" label="原材料" /><el-table-column prop="origin" label="原料产地" />
        <el-table-column prop="supplier" label="供应商负责人" /><el-table-column prop="factory" label="榨油厂负责人" />
        <el-table-column label="状态" width="150"><template slot-scope="{row}"><el-tag :type="tagType(row.status)">{{ statusName(row.status) }}</el-tag></template></el-table-column>
        <el-table-column prop="updated_at" label="最近更新时间" width="180" /><el-table-column label="操作" width="100"><template slot-scope="{row}"><el-button type="text" @click.stop="open(row)">完整追溯</el-button></template></el-table-column>
      </el-table>
    </div>

    <template v-else>
      <div class="trace-banner">
        <div><span>当前追溯批次</span><b>{{ detail.batch.trace_code }}</b><small>{{ detail.batch.material_name }} · {{ detail.batch.origin }}</small></div>
        <el-tag :type="tagType(detail.batch.status)" effect="dark">{{ statusName(detail.batch.status) }}</el-tag>
      </div>

      <section class="trace-section">
        <div class="section-heading"><span>01</span><div><h3>原料信息</h3><p>原料供应商提交的批次基础信息与质量检测结果</p></div></div>
        <div class="info-grid">
          <info-card label="原料名称" :value="detail.batch.material_name" />
          <info-card label="原料产地" :value="detail.batch.origin" />
          <info-card label="原料数量" :value="detail.batch.quantity+' '+detail.batch.unit" />
          <info-card label="质量等级" :value="detail.batch.quality_grade" />
          <info-card label="生产日期" :value="detail.batch.production_date" />
          <info-card label="批次创建时间" :value="detail.batch.created_at" />
          <info-card label="供应商负责人" :value="detail.batch.supplier" />
          <info-card label="检测报告" :value="detail.batch.test_report" wide />
        </div>
      </section>

      <section class="trace-section">
        <div class="section-heading"><span>02</span><div><h3>食用油压榨加工信息</h3><p>榨油厂接收原料后记录的产品、工艺和检验信息</p></div></div>
        <div v-if="processing" class="info-grid">
          <info-card label="榨油厂负责人" :value="detail.batch.factory || '尚未接收'" />
          <info-card label="成品名称" :value="processing.product_name" />
          <info-card label="生产批次" :value="processing.production_batch" />
          <info-card label="加工时间" :value="processing.production_time" />
          <info-card label="压榨加工工艺" :value="processing.process" wide />
          <info-card label="质量检验结果" :value="processing.inspection" wide />
        </div>
        <el-empty v-else description="该批次尚未进入压榨加工环节" :image-size="80" />
      </section>

      <section class="trace-section">
        <div class="section-heading"><span>03</span><div><h3>运输信息</h3><p>运输任务、国内运输路线、GPS轨迹及温湿度记录</p></div></div>
        <template v-if="detail.transport && detail.transport.id">
          <div class="info-grid">
            <info-card label="产品名称" :value="detail.transport.product_name" />
            <info-card label="产品数量" :value="detail.transport.product_quantity+' 吨'" />
            <info-card label="车辆牌照" :value="detail.transport.vehicle_no" />
            <info-card label="运输操作人" :value="detail.transport.transporter" />
            <info-card label="国内运输路线" :value="detail.transport.start_city+' → '+detail.transport.end_city" />
            <info-card label="任务创建时间" :value="detail.transport.created_at || '尚未创建'" />
            <info-card label="启运时间" :value="detail.transport.started_at || '尚未启运'" />
            <info-card label="运输完成时间" :value="detail.transport.completed_at || '尚未完成'" />
            <info-card label="运输说明" :value="detail.transport.note" wide />
          </div>
          <div class="route-summary">
            <div class="route-stats"><div><span>GPS节点</span><b>{{ detail.nodes.length }}</b></div><div><span>最高温度</span><b>{{ maxValue('temperature') }} ℃</b></div><div><span>最高湿度</span><b>{{ maxValue('humidity') }} %</b></div><div><span>路线类型</span><b>{{ routeType }}</b></div></div>
            <el-button type="primary" plain icon="el-icon-map-location" @click="toggleMap">{{ mapVisible ? '收起轨迹地图' : '查看轨迹地图' }}</el-button>
          </div>
          <div v-if="mapVisible" class="map-layout">
            <div class="map-card"><div ref="traceMap" class="trace-map" /><div v-if="mapError" class="map-error">{{ mapError }}</div></div>
            <el-table :data="detail.nodes" border max-height="320" empty-text="尚无运输轨迹"><el-table-column prop="seq" label="节点" width="65" /><el-table-column prop="longitude" label="经度" /><el-table-column prop="latitude" label="纬度" /><el-table-column prop="temperature" label="温度℃" /><el-table-column prop="humidity" label="湿度%" /><el-table-column prop="recorded_at" label="记录时间" width="180" /></el-table>
          </div>
        </template>
        <el-empty v-else description="该批次尚未安排运输任务" :image-size="80" />
      </section>

      <section class="trace-section">
        <div class="section-heading"><span>04</span><div><h3>零售信息</h3><p>零售方对到货产品进行核验后填写的收货与入库信息</p></div></div>
        <div v-if="receipt" class="info-grid">
          <info-card label="零售负责人" :value="detail.batch.retailer" />
          <info-card label="收货结果" :value="receipt.result" />
          <info-card label="实收数量" :value="receipt.quantity+' 吨'" />
          <info-card label="收货时间" :value="receipt.received_time" />
          <info-card label="入库仓库" :value="receipt.warehouse" />
          <info-card label="包装与铅封核验" :value="receipt.package_check || '包装与铅封完整'" />
          <info-card label="质量核验结果" :value="receipt.quality" wide />
        </div>
        <el-empty v-else description="该批次尚未完成零售收货" :image-size="80" />
      </section>

      <section class="trace-section timeline-section">
        <div class="section-heading"><span>05</span><div><h3>各环节可信时间线</h3><p>按执行时间展示每个环节的操作主体、业务结果与存证交易</p></div></div>
        <div v-if="detail.corrections.length||detail.rejections.length" class="exception-list">
          <el-alert v-for="(x,i) in detail.corrections" :key="'c'+i" :title="x.stage+'更正：'+x.reason" type="warning" :description="x.content+'；操作人：'+x.operator_name+'；时间：'+x.created_at" show-icon />
          <el-alert v-for="(x,i) in detail.rejections" :key="'r'+i" :title="x.stage+'退回：'+x.reason" type="error" :description="'操作人：'+x.operator_name+'；时间：'+x.created_at" show-icon />
        </div>
        <el-timeline><el-timeline-item v-for="(e,i) in detail.evidence" :key="i" :timestamp="String(e.created_at)" placement="top" color="#318f89"><el-card><div class="timeline-card"><div><h4>{{ e.business_type }}</h4><p>{{ e.business_summary }}</p><span>操作人：{{ e.operator_name }} · {{ e.operator_role }} · {{ e.operator_organization }}</span></div><el-popover trigger="click" width="500"><code class="full-hash">{{ e.transaction_hash }}</code><el-button slot="reference" size="mini">核验交易哈希</el-button></el-popover></div></el-card></el-timeline-item></el-timeline>
      </section>
    </template>
  </div>
</template>
<script>
import { traceBatches, traceDetail } from '@/api/business'
import { statusName, tagType } from '@/utils/workflow'
import { loadAMap } from '@/utils/amap'
const InfoCard = { functional: true, props: { label: String, value: [String, Number], wide: Boolean }, render(h, ctx) { return h('div', { class: ['info-card', { wide: ctx.props.wide }] }, [h('span', ctx.props.label), h('b', String(ctx.props.value || '暂无'))]) } }
export default {
  components: { InfoCard },
  data: () => ({ list: [], keyword: '', detail: null, mapVisible: false, map: null, mapError: '' }),
  computed: {
    filtered() { const k = this.keyword.trim(); return !k ? this.list : this.list.filter(x => [x.trace_code, x.material_name, x.origin].some(v => String(v || '').includes(k))) },
    processing() { return this.parseData(this.detail && this.detail.batch.processing_data) },
    receipt() { return this.parseData(this.detail && this.detail.batch.receipt_data) },
    routeType() { const n = this.detail ? this.detail.nodes.length : 0; return n <= 10 ? '市内短途' : n <= 14 ? '省内短途' : n <= 18 ? '跨省中长途' : '跨省长途' }
  },
  created() { traceBatches().then(r => { this.list = r.data; const code = this.$route.query.code; if (code) { const row = this.list.find(x => x.trace_code === code); if (row) this.open(row) } }) },
  beforeDestroy() { this.destroyMap() },
  methods: {
    statusName, tagType,
    open(row) { traceDetail(row.trace_code).then(r => { this.detail = r.data; this.mapVisible = false; this.destroyMap(); window.scrollTo({ top: 0, behavior: 'smooth' }) }) },
    backToList() { this.detail = null; this.mapVisible = false; this.destroyMap(); window.scrollTo({ top: 0, behavior: 'smooth' }) },
    toggleMap() {
      this.mapVisible = !this.mapVisible
      if (!this.mapVisible) return this.destroyMap()
      this.$nextTick(this.renderMap)
    },
    parseData(value) { if (!value) return null; if (typeof value === 'object') return Object.keys(value).length ? value : null; try { const result = JSON.parse(value); return result && Object.keys(result).length ? result : null } catch (e) { return null } },
    maxValue(key) { if (!this.detail || !this.detail.nodes.length) return '--'; return Math.max(...this.detail.nodes.map(x => Number(x[key]))).toFixed(1) },
    async renderMap() {
      this.destroyMap(); this.mapError = ''
      if (!this.detail || !this.detail.transport || !this.detail.transport.id || !this.$refs.traceMap) return
      try {
        const AMap = await loadAMap(); const t = this.detail.transport
        const start = [Number(t.start_lng), Number(t.start_lat)]; const end = [Number(t.end_lng), Number(t.end_lat)]; const nodes = this.detail.nodes.map(x => [Number(x.longitude), Number(x.latitude)]); const path = [start, ...nodes, end]
        this.map = new AMap.Map(this.$refs.traceMap, { viewMode: '2D', zoom: 7, center: start, mapStyle: 'amap://styles/whitesmoke', resizeEnable: true })
        this.map.addControl(new AMap.Scale()); this.map.addControl(new AMap.ToolBar({ position: 'RT' }))
        const line = new AMap.Polyline({ path, strokeColor: '#2b8d88', strokeWeight: 6, strokeOpacity: 0.9, showDir: true })
        const startMarker = this.marker(AMap, start, '起', '#67c23a'); const endMarker = this.marker(AMap, end, '终', '#f56c6c')
        const points = nodes.map(point => new AMap.CircleMarker({ center: point, radius: 5, strokeColor: '#fff', strokeWeight: 2, fillColor: '#409eff', fillOpacity: 1 }))
        this.map.add([line, startMarker, endMarker, ...points]); this.map.setFitView([line, startMarker, endMarker], false, [45, 45, 45, 45], 11)
      } catch (e) { this.mapError = `地图加载失败：${e.message || '请检查地图配置'}` }
    },
    marker(AMap, position, label, color) { return new AMap.Marker({ position, anchor: 'center', content: `<div style="width:30px;height:30px;border-radius:50%;background:${color};color:#fff;border:3px solid #fff;display:flex;align-items:center;justify-content:center;font-weight:700;box-shadow:0 2px 8px rgba(0,0,0,.25)">${label}</div>` }) },
    destroyMap() { if (this.map) { this.map.destroy(); this.map = null } }
  }
}
</script>
<style scoped>
.page-head{display:flex;align-items:center;justify-content:space-between}.toolbar{display:flex;justify-content:space-between;align-items:center;margin-bottom:16px;color:#687782}.trace-banner{display:flex;align-items:center;justify-content:space-between;padding:15px 20px;color:#fff;background:linear-gradient(120deg,#27566d,#318f89);border-radius:10px}.trace-banner span,.trace-banner b,.trace-banner small{display:block}.trace-banner b{margin:4px 0;font-size:21px}.trace-banner small{color:#d3e8e7}.trace-section{margin-top:14px;padding:18px 20px;background:#fff;border-radius:10px;box-shadow:0 2px 12px rgba(24,54,78,.06)}.section-heading{display:flex;align-items:center;margin-bottom:14px}.section-heading>span{display:flex;align-items:center;justify-content:center;width:34px;height:34px;margin-right:11px;color:#fff;background:#318f89;border-radius:50%;font-size:13px;font-weight:700}.section-heading h3{margin:0;color:#244756}.section-heading p{margin:3px 0 0;color:#8798a1;font-size:12px}.info-grid{display:grid;grid-template-columns:repeat(4,1fr);gap:9px}.info-card{padding:11px 13px;border:1px solid #e5edf0;border-radius:7px;background:#fafcfd;min-height:62px}.info-card.wide{grid-column:span 2}.info-card span,.info-card b{display:block}.info-card span{color:#8a9aa3;font-size:12px}.info-card b{margin-top:6px;color:#304f5f;line-height:1.4}.route-summary{display:flex;align-items:center;gap:16px;margin-top:14px}.route-stats{display:grid;grid-template-columns:repeat(4,1fr);gap:9px;flex:1}.route-stats div{display:flex;flex-direction:column;justify-content:center;padding:10px 12px;background:#f3f8f8;border-radius:8px}.route-stats span{color:#84959e;font-size:12px}.route-stats b{margin-top:4px;color:#285464;font-size:15px}.map-layout{display:grid;grid-template-columns:1.2fr 1fr;gap:12px;margin-top:14px}.map-card{position:relative;min-height:320px;border:1px solid #dce7ea;border-radius:8px;overflow:hidden}.trace-map{height:320px}.map-error{position:absolute;inset:0;display:flex;align-items:center;justify-content:center;color:#8b9ba4;background:#f3f7f8}.exception-list .el-alert{margin-bottom:8px}.timeline-section .el-timeline{padding-left:8px}.timeline-card{display:flex;align-items:center;justify-content:space-between}.timeline-card h4{margin:0 0 5px}.timeline-card p{margin:5px 0;color:#536c78}.timeline-card span{color:#8799a2;font-size:12px}.full-hash{display:block;word-break:break-all;padding:10px}@media(max-width:1200px){.info-grid{grid-template-columns:repeat(2,1fr)}.map-layout{grid-template-columns:1fr}.route-summary{align-items:stretch;flex-direction:column}.route-stats{grid-template-columns:repeat(4,1fr)}}
</style>
