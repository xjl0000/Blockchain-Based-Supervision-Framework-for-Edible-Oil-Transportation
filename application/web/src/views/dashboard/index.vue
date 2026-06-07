<template>
  <div class="page home-page">
    <section class="system-intro">
      <div class="intro-content">
        <span class="system-label">BLOCKCHAIN SUPERVISION PLATFORM</span>
        <h1>基于区块链的食用油运输监管系统</h1>
        <p>面向原材料供应、食用油压榨、运输流转、零售收货与监管追溯全过程，记录各参与主体的业务操作、运输轨迹、温湿度数据和可信存证。</p>
        <div class="intro-tags">
          <span><i class="el-icon-connection" /> 区块链可信存证</span>
          <span><i class="el-icon-location-information" /> GPS运输轨迹</span>
          <span><i class="el-icon-search" /> 全流程产品追溯</span>
        </div>
      </div>
    </section>

    <section class="business-section">
      <div class="section-title">
        <div><span>BUSINESS OVERVIEW</span><h2>目前系统业务量</h2></div>
        <p>以下数据按照当前登录角色可查看的业务范围统计</p>
      </div>
      <div class="business-grid">
        <div v-for="item in businessCards" :key="item.label" class="business-card">
          <div class="business-icon" :class="item.tone"><i :class="item.icon" /></div>
          <div><b>{{ item.value }}</b><span>{{ item.label }}</span><small>{{ item.description }}</small></div>
        </div>
      </div>
    </section>

    <section class="process-section">
      <div class="section-title">
        <div><span>BUSINESS PROCESS</span><h2>食用油运输业务流程</h2></div>
        <p>各环节业务数据按时间顺序形成完整、可追溯的可信记录</p>
      </div>
      <div class="process-flow">
        <template v-for="(item,index) in process">
          <div :key="item.title" class="process-card">
            <div class="process-number">0{{ index + 1 }}</div>
            <i :class="item.icon" />
            <h3>{{ item.title }}</h3>
            <p>{{ item.description }}</p>
            <span>{{ item.action }}</span>
          </div>
          <div v-if="index < process.length - 1" :key="'arrow'+index" class="process-arrow"><i class="el-icon-right" /></div>
        </template>
      </div>
    </section>
  </div>
</template>

<script>
import { dashboard } from '@/api/business'

export default {
  data: () => ({
    stats: { total: 0, pending: 0, moving: 0, completed: 0, evidence: 0 },
    process: [
      { title: '原材料供应', description: '创建原料批次并登记产地、数量、质量等级与检测报告。', action: '提交原料批次存证', icon: 'el-icon-document' },
      { title: '食用油压榨', description: '榨油厂接收原料，记录压榨工艺、成品批次与检验结果。', action: '提交加工生产存证', icon: 'el-icon-office-building' },
      { title: '运输流转', description: '运输人员接收任务，持续上传国内GPS轨迹及温湿度数据。', action: '形成运输过程存证', icon: 'el-icon-truck' },
      { title: '零售收货', description: '零售方核验数量、包装、质量和运输过程后确认入库。', action: '提交零售收货存证', icon: 'el-icon-shopping-bag-2' },
      { title: '监管追溯', description: '监管机构按溯源码核验参与主体、执行时间和完整存证链。', action: '查看全流程可信数据', icon: 'el-icon-search' }
    ]
  }),
  computed: {
    businessCards() {
      return [
        { label: '相关业务批次', value: this.stats.total, description: '当前角色可查看的批次总量', icon: 'el-icon-box', tone: 'blue' },
        { label: '待处理批次', value: this.stats.pending, description: '等待进入下一业务环节', icon: 'el-icon-time', tone: 'orange' },
        { label: '正在运输', value: this.stats.moving, description: '正在持续产生运输数据', icon: 'el-icon-truck', tone: 'cyan' },
        { label: '已完成批次', value: this.stats.completed, description: '已完成零售核验收货', icon: 'el-icon-circle-check', tone: 'green' },
        { label: '可信存证记录', value: this.stats.evidence, description: '相关批次存证记录总量', icon: 'el-icon-connection', tone: 'purple' }
      ]
    }
  },
  created() {
    dashboard().then(response => { this.stats = response.data })
  }
}
</script>

<style scoped>
.home-page{padding:26px}.system-intro{padding:34px 38px;background:#fff;border:1px solid #e2eaed;border-left:5px solid #318f89;border-radius:12px;box-shadow:0 2px 14px rgba(25,58,78,.07)}.intro-content{max-width:1000px}.system-label,.section-title span{font-size:11px;letter-spacing:1.8px}.system-label{color:#318f89}.system-intro h1{margin:11px 0 13px;color:#244555;font-size:30px}.system-intro p{margin:0;color:#667d88;line-height:1.8}.intro-tags{display:flex;gap:12px;margin-top:21px}.intro-tags span{padding:7px 12px;color:#39716f;border:1px solid #d5e6e5;border-radius:18px;background:#f5fafa;font-size:12px}.business-section,.process-section{margin-top:22px;padding:28px;background:#fff;border-radius:12px;box-shadow:0 2px 14px rgba(25,58,78,.07)}.section-title{display:flex;align-items:flex-end;justify-content:space-between;margin-bottom:24px}.section-title span{color:#318f89}.section-title h2{margin:7px 0 0;color:#244555;font-size:23px}.section-title p{margin:0;color:#8a9aa3;font-size:12px}.business-grid{display:grid;grid-template-columns:repeat(5,1fr);gap:15px}.business-card{display:flex;align-items:center;padding:19px;border:1px solid #e6edf0;border-radius:9px;background:#fbfcfd}.business-icon{display:flex;align-items:center;justify-content:center;width:48px;height:48px;margin-right:13px;color:#409eff;background:#edf5ff;border-radius:10px;font-size:23px}.business-icon.orange{color:#e6a23c;background:#fff6e8}.business-icon.cyan{color:#24a5a0;background:#eaf8f7}.business-icon.green{color:#67c23a;background:#f0f9eb}.business-icon.purple{color:#8c70c7;background:#f4f0fb}.business-card b,.business-card span,.business-card small{display:block}.business-card b{color:#274858;font-size:26px}.business-card span{margin:3px 0;color:#516a76}.business-card small{color:#98a5ac;font-size:11px}.process-flow{display:flex;align-items:stretch}.process-card{position:relative;flex:1;padding:21px 18px;border:1px solid #e4ecef;border-radius:10px;background:#fbfcfd}.process-number{position:absolute;right:12px;top:10px;color:#dce8eb;font-size:22px;font-weight:700}.process-card>i{font-size:27px;color:#318f89}.process-card h3{margin:14px 0 8px;color:#294c5c}.process-card p{min-height:64px;margin:0;color:#788b95;font-size:12px;line-height:1.7}.process-card span{display:block;margin-top:13px;color:#318f89;font-size:11px}.process-arrow{display:flex;align-items:center;justify-content:center;width:38px;color:#8ab8b5;font-size:20px}@media(max-width:1200px){.business-grid{grid-template-columns:repeat(3,1fr)}.process-flow{display:grid;grid-template-columns:repeat(2,1fr);gap:12px}.process-arrow{display:none}}
</style>
