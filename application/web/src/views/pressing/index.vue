<template>
  <div class="page">
    <div class="page-head">
      <div><h2>食用油压榨管理</h2><p>榨油厂在此接收原料、填写压榨加工信息，并为加工完成的食用油发起运输任务。</p></div>
      <el-button type="primary" icon="el-icon-truck" @click="openTask">发起运输任务</el-button>
    </div>
    <el-table :data="list" border>
      <el-table-column prop="trace_code" label="溯源码" width="175" />
      <el-table-column prop="material_name" label="原材料" />
      <el-table-column prop="origin" label="原料产地" />
      <el-table-column prop="supplier" label="供应商负责人" />
      <el-table-column label="数量"><template slot-scope="{row}">{{ row.quantity }} {{ row.unit }}</template></el-table-column>
      <el-table-column label="状态" width="150"><template slot-scope="{row}"><el-tag :type="tagType(row.status)">{{ statusName(row.status) }}</el-tag></template></el-table-column>
      <el-table-column label="操作" min-width="300"><template slot-scope="{row}">
        <el-button size="mini" @click="detail(row)">查看信息</el-button>
        <el-button v-if="row.status==='pending_factory'" size="mini" type="success" @click="decision(row,true)">接收原料</el-button>
        <el-button v-if="row.status==='pending_factory'" size="mini" type="danger" @click="decision(row,false)">拒收退回</el-button>
        <el-button v-if="row.status==='factory_received'" size="mini" type="primary" @click="process(row)">填写压榨信息</el-button>
        <el-button v-if="row.status==='processed'" size="mini" type="warning" @click="openTask(row)">安排运输</el-button>
        <el-button v-if="row.factory_id && row.status!=='factory_received'" size="mini" @click="correct(row)">追加更正</el-button>
      </template></el-table-column>
    </el-table>

    <el-dialog title="压榨加工信息" :visible.sync="processVisible" width="620px">
      <el-form label-width="110px">
        <el-form-item label="产品名称"><el-input v-model="processing.product_name" /></el-form-item>
        <el-form-item label="生产批次"><el-input v-model="processing.production_batch" /></el-form-item>
        <el-form-item label="压榨工艺"><el-input v-model="processing.process" /></el-form-item>
        <el-form-item label="加工时间"><el-date-picker v-model="processing.production_time" type="datetime" value-format="yyyy-MM-dd HH:mm:ss" /></el-form-item>
        <el-form-item label="质量检验"><el-input v-model="processing.inspection" type="textarea" /></el-form-item>
      </el-form>
      <span slot="footer"><el-button @click="processVisible=false">取消</el-button><el-button type="primary" @click="saveProcess">提交压榨存证</el-button></span>
    </el-dialog>

    <el-dialog title="发起食用油运输任务" :visible.sync="taskVisible" width="720px">
      <el-form label-width="110px">
        <el-form-item label="加工批次"><el-select v-model="task.batchID"><el-option v-for="b in available" :key="b.id" :label="b.trace_code+' / '+b.material_name" :value="b.id" /></el-select></el-form-item>
        <el-row :gutter="14">
          <el-col :span="12"><el-form-item label="运输人员"><el-select v-model="task.transporterID"><el-option v-for="x in drivers" :key="x.id" :label="x.name+' / '+x.organization" :value="x.id" /></el-select></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="目标零售商"><el-select v-model="task.retailerID"><el-option v-for="x in retailers" :key="x.id" :label="x.name+' / '+x.organization" :value="x.id" /></el-select></el-form-item></el-col>
        </el-row>
        <el-row :gutter="14"><el-col :span="12"><el-form-item label="司机姓名"><el-input v-model="task.driverName" /></el-form-item></el-col><el-col :span="12"><el-form-item label="车辆牌照"><el-input v-model="task.vehicleNo" /></el-form-item></el-col></el-row>
        <el-row :gutter="14"><el-col :span="12"><el-form-item label="产品名称"><el-input v-model="task.productName" /></el-form-item></el-col><el-col :span="12"><el-form-item label="产品数量"><el-input-number v-model="task.productQuantity" :min="0" /></el-form-item></el-col></el-row>
        <el-row :gutter="14"><el-col :span="12"><el-form-item label="起点城市"><el-select v-model="startKey" @change="setStart"><el-option v-for="(v,k) in cities" :key="k" :label="k" :value="k" /></el-select></el-form-item></el-col><el-col :span="12"><el-form-item label="目的城市"><el-select v-model="endKey" @change="setEnd"><el-option v-for="(v,k) in cities" :key="k" :label="k" :value="k" /></el-select></el-form-item></el-col></el-row>
        <el-form-item label="运输说明"><el-input v-model="task.note" type="textarea" /></el-form-item>
      </el-form>
      <span slot="footer"><el-button @click="taskVisible=false">取消</el-button><el-button type="primary" @click="saveTask">提交运输任务</el-button></span>
    </el-dialog>

    <el-dialog title="批次信息" :visible.sync="detailVisible" width="700px">
      <el-descriptions v-if="selected" :column="2" border>
        <el-descriptions-item label="溯源码">{{ selected.trace_code }}</el-descriptions-item><el-descriptions-item label="状态">{{ statusName(selected.status) }}</el-descriptions-item>
        <el-descriptions-item label="供应商">{{ selected.supplier }}</el-descriptions-item><el-descriptions-item label="原料产地">{{ selected.origin }}</el-descriptions-item>
        <el-descriptions-item label="质量等级">{{ selected.quality_grade }}</el-descriptions-item><el-descriptions-item label="检测报告">{{ selected.test_report }}</el-descriptions-item>
        <el-descriptions-item label="加工信息" :span="2"><pre>{{ pretty(selected.processing_data) }}</pre></el-descriptions-item>
      </el-descriptions>
    </el-dialog>
  </div>
</template>

<script>
import { batches, factoryDecision, submitProcessing, addCorrection, roleOptions, createTransport } from '@/api/business'
import { statusName, tagType } from '@/utils/workflow'
const cities = { 哈尔滨市: [126.642464, 45.756967, '黑龙江省'], 北京市: [116.407526, 39.90403, '北京市'], 上海市: [121.473701, 31.230416, '上海市'], 青岛市: [120.38264, 36.067082, '山东省'], 郑州市: [113.625368, 34.7466, '河南省'] }
export default {
  data: () => ({ list: [], available: [], drivers: [], retailers: [], selected: null, detailVisible: false, processVisible: false, taskVisible: false, processing: {}, startKey: '哈尔滨市', endKey: '北京市', cities, task: { driverName: '王志强', vehicleNo: '黑A·E6608', productName: '一级压榨食用油', productQuantity: 18.6, note: '食品级专用罐车运输，全程铅封并记录定位及温湿度' }}),
  created() { this.load() },
  methods: {
    statusName, tagType,
    load() { batches().then(r => { this.list = r.data; this.available = r.data.filter(x => x.status === 'processed') }); roleOptions('运输人员').then(r => { this.drivers = r.data }); roleOptions('零售商').then(r => { this.retailers = r.data }) },
    decision(row, accept) { this.$prompt(accept ? '填写接收核验备注' : '填写拒收原因', '原料接收核验', { inputValue: accept ? '数量与质量检测核验通过' : '' }).then(x => factoryDecision({ id: row.id, accept, reason: x.value })).then(x => { this.$message.success(x.message); this.load() }) },
    process(row) { this.selected = row; this.processing = { product_name: '一级压榨食用油', production_batch: row.trace_code + '-P', process: '原料筛选、低温压榨、物理精炼、灌装封装', production_time: '', inspection: '酸价、过氧化值、溶剂残留量检验合格' }; this.processVisible = true },
    saveProcess() { submitProcessing({ id: this.selected.id, data: this.processing }).then(x => { this.$message.success(x.message); this.processVisible = false; this.load() }) },
    openTask(row) { if (row) this.task.batchID = row.id; this.setStart(this.startKey); this.setEnd(this.endKey); this.taskVisible = true },
    setStart(k) { const p = cities[k]; Object.assign(this.task, { startCity: k, startProvince: p[2], startLng: p[0], startLat: p[1] }) },
    setEnd(k) { const p = cities[k]; Object.assign(this.task, { endCity: k, endProvince: p[2], endLng: p[0], endLat: p[1] }) },
    saveTask() { createTransport(this.task).then(x => { this.$message.success(x.message); this.taskVisible = false; this.load() }) },
    correct(row) { this.$prompt('填写更正原因与内容', '追加更正').then(x => addCorrection({ batchID: row.id, stage: '压榨加工', reason: x.value, content: x.value })).then(x => this.$message.success(x.message)) },
    detail(row) { this.selected = row; this.detailVisible = true },
    pretty(v) { if (!v) return '暂无'; try { return JSON.stringify(typeof v === 'string' ? JSON.parse(v) : v, null, 2) } catch (e) { return v } }
  }
}
</script>
