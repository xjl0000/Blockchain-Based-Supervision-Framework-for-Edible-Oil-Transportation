<template>
  <div class="page">
    <div class="page-head"><div><h2>区块链存证查询</h2><p>先按溯源码选择批次，再查看该批次按时间顺序形成的完整存证链。</p></div></div>
    <div class="panel">
      <div class="toolbar"><el-input v-model="keyword" clearable placeholder="搜索溯源码" prefix-icon="el-icon-search" style="width:300px" /><span>可查看 {{ groups.length }} 个批次的存证</span></div>
      <el-table :data="filtered" border highlight-current-row @row-click="select">
        <el-table-column prop="trace_code" label="溯源码" width="190" />
        <el-table-column prop="count" label="存证数量" width="110" />
        <el-table-column label="上链状态" width="150"><template slot-scope="{row}"><el-tag :type="statusTag(row.chain_stats)" size="mini" effect="dark">{{ row.chain_stats }}</el-tag></template></el-table-column>
        <el-table-column prop="first_time" label="首条存证时间" width="190" />
        <el-table-column prop="latest_time" label="最新存证时间" width="190" />
        <el-table-column prop="latest_type" label="最新存证环节" />
        <el-table-column label="操作" width="120"><template slot-scope="{row}"><el-button type="text" @click.stop="select(row)">查看存证链</el-button></template></el-table-column>
      </el-table>
    </div>
    <div v-if="selected" class="panel">
      <div class="panel-title">{{ selected.trace_code }} · 完整存证链</div>
      <el-table :data="selected.records" border>
        <el-table-column type="index" label="序号" width="60" />
        <el-table-column prop="business_type" label="存证类型" width="150" />
        <el-table-column prop="business_summary" label="业务摘要" />
        <el-table-column label="操作主体" width="190"><template slot-scope="{row}">{{ row.operator_name }}<br><small>{{ row.operator_role }}</small></template></el-table-column>
        <el-table-column prop="created_at" label="存证时间" width="180" />
        <el-table-column label="上链状态" width="130">
          <template slot-scope="{row}">
            <el-tag :type="chainStatusType(row.fabric_status)" size="mini" effect="dark">{{ chainStatusLabel(row.fabric_status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="Fabric TxID" width="180">
          <template slot-scope="{row}">
            <span v-if="row.fabric_tx_id" class="tx-id" :title="row.fabric_tx_id">{{ row.fabric_tx_id.substring(0,16) }}...</span>
            <span v-else class="no-data">-</span>
          </template>
        </el-table-column>
        <el-table-column label="区块号" width="90">
          <template slot-scope="{row}">
            <span v-if="row.fabric_block_number">{{ row.fabric_block_number }}</span>
            <span v-else class="no-data">-</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="180">
          <template slot-scope="{row}">
            <el-button v-if="row.fabric_status==='confirmed'" size="mini" type="primary" plain @click.stop="verify(row)">哈希核验</el-button>
            <el-button v-if="row.fabric_status==='failed'" size="mini" type="warning" plain @click.stop="retry(row)">重新上链</el-button>
            <el-popover v-if="row.fabric_tx_id" trigger="click" width="520">
              <div class="hash">
                <b>事件 ID</b><code>{{ row.event_id }}</code>
                <b>数据哈希</b><code>{{ row.data_hash }}</code>
                <b>前置哈希</b><code>{{ row.previous_hash }}</code>
                <b>Fabric TxID</b><code>{{ row.fabric_tx_id }}</code>
                <b>区块号</b><code>{{ row.fabric_block_number }}</code>
                <b>确认时间</b><code>{{ row.confirmed_at || '尚未确认' }}</code>
              </div>
              <el-button slot="reference" size="mini">查看详情</el-button>
            </el-popover>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 核验结果弹窗 -->
    <el-dialog title="哈希核验结果" :visible.sync="verifyVisible" width="560px">
      <div v-if="verifyResult" class="verify-result">
        <div :class="['verify-badge', verifyResult.verify_valid ? 'valid' : 'invalid']">
          <i :class="verifyResult.verify_valid ? 'el-icon-success' : 'el-icon-warning'" />
          <span>{{ verifyResult.verify_status }}</span>
        </div>
        <div class="verify-detail">
          <div><label>链上哈希</label><code>{{ verifyResult.chain_hash || '-' }}</code></div>
          <div><label>当前哈希</label><code>{{ verifyResult.current_hash || '-' }}</code></div>
          <div><label>Fabric TxID</label><code>{{ verifyResult.fabric_tx_id || '-' }}</code></div>
          <div><label>事件 ID</label><code>{{ verifyResult.event_id || '-' }}</code></div>
        </div>
      </div>
    </el-dialog>
  </div>
</template>
<script>
import { evidence, verifyEvidence, retryEvidence } from '@/api/business'
import { mapGetters } from 'vuex'
export default {
  data: () => ({ groups: [], selected: null, keyword: '', verifyVisible: false, verifyResult: null }),
  computed: {
    ...mapGetters(['userType']),
    filtered() { const k = this.keyword.trim(); return !k ? this.groups : this.groups.filter(x => x.trace_code.includes(k)) }
  },
  created() {
    evidence().then(r => {
      const map = {}
      r.data.forEach(x => {
        if (!map[x.trace_code]) map[x.trace_code] = { trace_code: x.trace_code, records: [] }
        map[x.trace_code].records.push(x)
      })
      this.groups = Object.values(map).map(x => {
        const confirmed = x.records.filter(r => r.fabric_status === 'confirmed').length
        const failed = x.records.filter(r => r.fabric_status === 'failed').length
        let stats = `${confirmed}/${x.records.length} 已上链`
        if (failed > 0) stats += ` (${failed}失败)`
        return { ...x, count: x.records.length, first_time: x.records[0].created_at, latest_time: x.records[x.records.length - 1].created_at, latest_type: x.records[x.records.length - 1].business_type, chain_stats: stats }
      })
    })
  },
  methods: {
    select(row) { this.selected = row; this.$nextTick(() => window.scrollTo({ top: 450, behavior: 'smooth' })) },
    chainStatusType(status) { return { confirmed: 'success', pending: 'warning', failed: 'danger', submitting: 'info' }[status] || 'info' },
    chainStatusLabel(status) { return { confirmed: '已上链', pending: '待上链', failed: '上链失败', submitting: '提交中' }[status] || status },
    statusTag(stats) { if (stats.includes('失败')) return 'warning'; return 'success' },
    verify(row) {
      const loading = this.$loading({ text: '正在核验链上数据...' })
      verifyEvidence(row.id).then(r => {
        loading.close()
        this.verifyResult = r.data
        this.verifyVisible = true
      }).catch(err => {
        loading.close()
        this.$message.error('核验失败：' + (err.message || '未知错误'))
      })
    },
    retry(row) {
      this.$confirm('确认重新将此记录提交到区块链网络？', '重新上链', { type: 'warning' }).then(() => {
        const loading = this.$loading({ text: '正在提交到区块链...' })
        retryEvidence(row.id).then(r => {
          loading.close()
          this.$message.success(`上链成功！TxID: ${r.tx_id}`)
          row.fabric_status = 'confirmed'
          row.fabric_tx_id = r.tx_id
          row.fabric_block_number = r.block_number
        }).catch(err => {
          loading.close()
          this.$message.error('上链失败：' + (err.message || '未知错误'))
        })
      }).catch(() => {})
    }
  }
}
</script>
<style scoped>
.toolbar{display:flex;justify-content:space-between;align-items:center;margin-bottom:16px;color:#687782}
.hash{display:grid;gap:6px}.hash code{word-break:break-all;padding:7px;background:#f4f7f9;display:block}
.tx-id{font-family:monospace;font-size:12px;color:#318f89;cursor:help}
.no-data{color:#c0c4cc}
.verify-result{text-align:center}
.verify-badge{display:inline-flex;align-items:center;gap:10px;padding:16px 32px;border-radius:12px;font-size:18px;font-weight:600;margin-bottom:20px}
.verify-badge.valid{background:#f0f9eb;color:#67c23a}
.verify-badge.invalid{background:#fef0f0;color:#f56c6c}
.verify-badge i{font-size:28px}
.verify-detail{text-align:left}
.verify-detail div{display:grid;grid-template-columns:100px 1fr;gap:8px;padding:8px 0;border-bottom:1px solid #f0f2f5}
.verify-detail label{color:#8a9aa3;font-size:13px}
.verify-detail code{word-break:break-all;font-size:12px;background:#f4f7f9;padding:4px 8px;border-radius:4px}
</style>
