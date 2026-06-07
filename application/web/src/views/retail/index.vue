<template>
  <div class="page">
    <div class="page-head"><div><h2>零售管理</h2><p>零售方在此核验到货产品、运输数据并填写收货与入库信息。</p></div></div>
    <el-table :data="list" border>
      <el-table-column prop="trace_code" label="溯源码" width="175" /><el-table-column prop="product_name" label="产品名称" />
      <el-table-column label="数量"><template slot-scope="{row}">{{ row.product_quantity }} 吨</template></el-table-column>
      <el-table-column prop="factory" label="榨油厂负责人" /><el-table-column prop="transporter" label="运输人员" /><el-table-column prop="vehicle_no" label="车辆" />
      <el-table-column label="状态" width="140"><template slot-scope="{row}"><el-tag :type="tagType(row.status)">{{ statusName(row.status) }}</el-tag></template></el-table-column>
      <el-table-column label="操作" min-width="260"><template slot-scope="{row}">
        <el-button size="mini" @click="showDetail(row)">查看运输信息</el-button>
        <el-button v-if="row.status==='pending_retail'" size="mini" type="success" @click="openReceipt(row)">填写收货信息</el-button>
        <el-button v-if="row.status==='pending_retail'" size="mini" type="danger" @click="reject(row)">拒收退回</el-button>
      </template></el-table-column>
    </el-table>
    <el-dialog title="零售收货与入库信息" :visible.sync="receiptVisible" width="620px">
      <el-form label-width="115px">
        <el-form-item label="实收数量"><el-input-number v-model="receipt.quantity" :min="0" /> 吨</el-form-item>
        <el-form-item label="收货时间"><el-date-picker v-model="receipt.received_time" type="datetime" value-format="yyyy-MM-dd HH:mm:ss" /></el-form-item>
        <el-form-item label="入库仓库"><el-input v-model="receipt.warehouse" /></el-form-item>
        <el-form-item label="包装与铅封"><el-input v-model="receipt.package_check" /></el-form-item>
        <el-form-item label="质量核验结果"><el-input v-model="receipt.quality" type="textarea" /></el-form-item>
      </el-form>
      <span slot="footer"><el-button @click="receiptVisible=false">取消</el-button><el-button type="primary" @click="confirmReceipt">确认收货并存证</el-button></span>
    </el-dialog>
    <el-dialog title="运输与到货信息" :visible.sync="detailVisible" width="720px">
      <el-descriptions v-if="selected" :column="2" border>
        <el-descriptions-item label="溯源码">{{ selected.trace_code }}</el-descriptions-item><el-descriptions-item label="产品">{{ selected.product_name }}</el-descriptions-item>
        <el-descriptions-item label="运输人员">{{ selected.transporter }}</el-descriptions-item><el-descriptions-item label="车辆">{{ selected.vehicle_no }}</el-descriptions-item>
        <el-descriptions-item label="运输路线">{{ selected.start_city }} → {{ selected.end_city }}</el-descriptions-item><el-descriptions-item label="任务创建时间">{{ selected.created_at }}</el-descriptions-item>
        <el-descriptions-item label="运输说明" :span="2">{{ selected.note }}</el-descriptions-item>
      </el-descriptions>
    </el-dialog>
  </div>
</template>
<script>
import { transports, retailDecision } from '@/api/business'
import { statusName, tagType } from '@/utils/workflow'
export default {
  data: () => ({ list: [], selected: null, receiptVisible: false, detailVisible: false, receipt: {}}),
  created() { this.load() },
  methods: {
    statusName, tagType, load() { transports().then(r => { this.list = r.data }) },
    openReceipt(row) { this.selected = row; this.receipt = { result: '确认收货', quantity: row.product_quantity, received_time: '', warehouse: '食品级成品油专用仓', package_check: '包装与铅封完整', quality: '数量一致、包装完整、质量抽检合格' }; this.receiptVisible = true },
    confirmReceipt() { retailDecision({ batchID: this.selected.batch_id, accept: true, reason: this.receipt.quality, data: this.receipt }).then(x => { this.$message.success(x.message); this.receiptVisible = false; this.load() }) },
    reject(row) { this.$prompt('填写拒收退回原因', '零售收货核验').then(x => retailDecision({ batchID: row.batch_id, accept: false, reason: x.value, data: {}})).then(x => { this.$message.success(x.message); this.load() }) },
    showDetail(row) { this.selected = row; this.detailVisible = true }
  }
}
</script>
