<template>
  <div class="page">
    <div class="page-head"><div><h2>区块链存证查询</h2><p>先按溯源码选择批次，再查看该批次按时间顺序形成的完整存证链。</p></div></div>
    <div class="panel">
      <div class="toolbar"><el-input v-model="keyword" clearable placeholder="搜索溯源码" prefix-icon="el-icon-search" style="width:300px" /><span>可查看 {{ groups.length }} 个批次的存证</span></div>
      <el-table :data="filtered" border highlight-current-row @row-click="select">
        <el-table-column prop="trace_code" label="溯源码" width="190" /><el-table-column prop="count" label="存证数量" width="110" />
        <el-table-column prop="first_time" label="首条存证时间" width="190" /><el-table-column prop="latest_time" label="最新存证时间" width="190" />
        <el-table-column prop="latest_type" label="最新存证环节" /><el-table-column label="操作" width="120"><template slot-scope="{row}"><el-button type="text" @click.stop="select(row)">查看存证链</el-button></template></el-table-column>
      </el-table>
    </div>
    <div v-if="selected" class="panel">
      <div class="panel-title">{{ selected.trace_code }} · 完整存证链</div>
      <el-table :data="selected.records" border>
        <el-table-column type="index" label="序号" width="60" /><el-table-column prop="business_type" label="存证类型" width="150" /><el-table-column prop="business_summary" label="业务摘要" />
        <el-table-column label="操作主体" width="190"><template slot-scope="{row}">{{ row.operator_name }}<br><small>{{ row.operator_role }}</small></template></el-table-column>
        <el-table-column prop="created_at" label="存证时间" width="180" /><el-table-column label="哈希核验" width="180"><template slot-scope="{row}"><el-popover trigger="click" width="520"><div class="hash"><b>数据哈希</b><code>{{ row.data_hash }}</code><b>前置哈希</b><code>{{ row.previous_hash }}</code><b>交易哈希</b><code>{{ row.transaction_hash }}</code><b>区块哈希</b><code>{{ row.block_hash }}</code></div><el-button slot="reference" size="mini">查看完整哈希</el-button></el-popover></template></el-table-column>
      </el-table>
    </div>
  </div>
</template>
<script>
import { evidence } from '@/api/business'
export default {
  data: () => ({ groups: [], selected: null, keyword: '' }),
  computed: { filtered() { const k = this.keyword.trim(); return !k ? this.groups : this.groups.filter(x => x.trace_code.includes(k)) } },
  created() { evidence().then(r => { const map = {}; r.data.forEach(x => { if (!map[x.trace_code]) map[x.trace_code] = { trace_code: x.trace_code, records: [] }; map[x.trace_code].records.push(x) }); this.groups = Object.values(map).map(x => ({ ...x, count: x.records.length, first_time: x.records[0].created_at, latest_time: x.records[x.records.length - 1].created_at, latest_type: x.records[x.records.length - 1].business_type })) }) },
  methods: { select(row) { this.selected = row; this.$nextTick(() => window.scrollTo({ top: 450, behavior: 'smooth' })) } }
}
</script>
<style scoped>.toolbar{display:flex;justify-content:space-between;align-items:center;margin-bottom:16px;color:#687782}.hash{display:grid;gap:6px}.hash code{word-break:break-all;padding:7px;background:#f4f7f9}</style>
