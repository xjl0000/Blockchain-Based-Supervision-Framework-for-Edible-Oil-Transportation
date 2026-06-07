<template>
  <div class="page">
    <div class="page-head"><div><h2>原材料管理</h2><p>原料供应商创建、维护并提交原材料批次，正式提交后生成存证。</p></div><el-button type="primary" icon="el-icon-plus" @click="openCreate">创建原料批次</el-button></div>
    <el-table :data="list" border>
      <el-table-column prop="trace_code" label="唯一溯源码" width="180" />
      <el-table-column prop="material_name" label="原料名称" />
      <el-table-column prop="origin" label="原料产地" />
      <el-table-column label="数量"><template slot-scope="{row}">{{ row.quantity }} {{ row.unit }}</template></el-table-column>
      <el-table-column prop="quality_grade" label="质量等级" />
      <el-table-column label="流程状态" width="150"><template slot-scope="{row}"><el-tag :type="tagType(row.status)">{{ statusName(row.status) }}</el-tag></template></el-table-column>
      <el-table-column label="操作" min-width="250"><template slot-scope="{row}">
        <el-button size="mini" @click="showDetail(row)">详情</el-button>
        <template>
          <el-button v-if="['raw_draft','returned_supplier'].includes(row.status)" size="mini" type="primary" @click="edit(row)">编辑原料信息</el-button>
          <el-button v-if="row.status==='raw_draft'" size="mini" type="danger" @click="remove(row)">删除草稿</el-button>
          <el-button v-if="['raw_draft','returned_supplier'].includes(row.status)" size="mini" type="success" @click="submit(row)">提交存证</el-button>
        </template>
        <el-button v-if="!['raw_draft'].includes(row.status)" size="mini" type="warning" @click="correct(row)">追加更正</el-button>
      </template></el-table-column>
    </el-table>
    <el-dialog :title="form.id?'编辑原料草稿':'创建原料批次'" :visible.sync="formVisible" width="600px"><el-form label-width="110px">
      <el-form-item label="原料名称"><el-input v-model="form.materialName" /></el-form-item><el-form-item label="原料产地"><el-input v-model="form.origin" /></el-form-item>
      <el-form-item label="原料数量"><el-input-number v-model="form.quantity" :min="0" /><el-select v-model="form.unit" style="width:100px;margin-left:10px"><el-option label="吨" value="吨" /><el-option label="千克" value="千克" /></el-select></el-form-item>
      <el-form-item label="质量等级"><el-input v-model="form.qualityGrade" /></el-form-item><el-form-item label="生产日期"><el-date-picker v-model="form.productionDate" value-format="yyyy-MM-dd" /></el-form-item><el-form-item label="检测报告摘要"><el-input v-model="form.testReport" type="textarea" /></el-form-item>
    </el-form><span slot="footer"><el-button @click="formVisible=false">取消</el-button><el-button type="primary" @click="save">保存草稿</el-button></span></el-dialog>
    <el-dialog title="加工生产信息录入" :visible.sync="processVisible" width="600px"><el-form label-width="110px"><el-form-item label="产品名称"><el-input v-model="processing.product_name" /></el-form-item><el-form-item label="产品批次"><el-input v-model="processing.production_batch" /></el-form-item><el-form-item label="加工工艺"><el-input v-model="processing.process" /></el-form-item><el-form-item label="加工时间"><el-date-picker v-model="processing.production_time" type="datetime" value-format="yyyy-MM-dd HH:mm:ss" /></el-form-item><el-form-item label="质量检验"><el-input v-model="processing.inspection" type="textarea" /></el-form-item></el-form><span slot="footer"><el-button @click="processVisible=false">取消</el-button><el-button type="primary" @click="saveProcess">提交加工存证</el-button></span></el-dialog>
    <el-dialog title="批次详细信息" :visible.sync="detailVisible" width="680px"><el-descriptions v-if="selected" :column="2" border><el-descriptions-item label="溯源码">{{ selected.trace_code }}</el-descriptions-item><el-descriptions-item label="状态">{{ statusName(selected.status) }}</el-descriptions-item><el-descriptions-item label="供应商">{{ selected.supplier }}</el-descriptions-item><el-descriptions-item label="榨油厂">{{ selected.factory||'待接收' }}</el-descriptions-item><el-descriptions-item label="原料">{{ selected.material_name }}</el-descriptions-item><el-descriptions-item label="产地">{{ selected.origin }}</el-descriptions-item><el-descriptions-item label="质量等级">{{ selected.quality_grade }}</el-descriptions-item><el-descriptions-item label="检测报告">{{ selected.test_report }}</el-descriptions-item><el-descriptions-item label="加工信息" :span="2"><pre>{{ pretty(selected.processing_data) }}</pre></el-descriptions-item></el-descriptions></el-dialog>
  </div>
</template>
<script>
import { mapGetters } from 'vuex'
import { batches, createBatch, updateBatch, deleteBatch, submitBatch, factoryDecision, submitProcessing, addCorrection } from '@/api/business'
import { statusName, tagType } from '@/utils/workflow'
export default { data: () => ({ list: [], formVisible: false, processVisible: false, detailVisible: false, selected: null, form: {}, processing: {}}), computed: { ...mapGetters(['role']) }, created() { this.load() }, methods: { statusName, tagType, load() { batches().then(r => { this.list = r.data }) }, openCreate() { this.form = { quantity: 20, unit: '吨' }; this.formVisible = true }, edit(r) { this.form = { id: r.id, materialName: r.material_name, origin: r.origin, quantity: r.quantity, unit: r.unit, qualityGrade: r.quality_grade, productionDate: r.production_date, testReport: r.test_report }; this.formVisible = true }, save() { const fn = this.form.id ? updateBatch : createBatch; fn(this.form).then(r => { this.$message.success(r.message); this.formVisible = false; this.load() }) }, remove(r) { this.$confirm('确认删除该未提交草稿？').then(() => deleteBatch(r.id)).then(x => { this.$message.success(x.message); this.load() }) }, submit(r) { this.$confirm('提交后原始信息将锁定并生成区块链存证，确认提交？').then(() => submitBatch(r.id)).then(x => { this.$message.success(x.message); this.load() }) }, decision(r, accept) { this.$prompt(accept ? '可填写接收备注' : '请填写拒收原因', '原料接收核验', { inputValue: accept ? '原料数量及质量核验通过' : '' }).then(x => factoryDecision({ id: r.id, accept, reason: x.value })).then(x => { this.$message.success(x.message); this.load() }) }, process(r) { this.selected = r; this.processing = { product_name: '一级压榨食用油', production_batch: r.trace_code + '-P', process: '低温压榨', production_time: '', inspection: '检验合格' }; this.processVisible = true }, saveProcess() { submitProcessing({ id: this.selected.id, data: this.processing }).then(x => { this.$message.success(x.message); this.processVisible = false; this.load() }) }, correct(r) { this.$prompt('请输入更正原因和更正内容，原始记录不会被覆盖', '追加更正记录').then(x => addCorrection({ batchId: r.id, stage: '业务数据', reason: x.value, content: x.value })).then(x => this.$message.success(x.message)) }, showDetail(r) { this.selected = r; this.detailVisible = true }, pretty(v) { if (!v) return '暂无'; try { return JSON.stringify(typeof v === 'string' ? JSON.parse(v) : v, null, 2) } catch (e) { return v } } }}
</script>
