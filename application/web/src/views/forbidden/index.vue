<template>
  <div class="forbidden-page">
    <div class="forbidden-card">
      <div class="lock"><i class="el-icon-lock" /></div>
      <h2>当前用户无权限</h2>
      <el-button type="primary" @click="$router.push('/dashboard')">返回主界面</el-button>
    </div>
  </div>
</template>
<script>export default { name: 'Forbidden' }</script>
<style scoped>
.forbidden-page{min-height:calc(100vh - 50px);display:flex;align-items:center;justify-content:center;background:#f4f7fa;padding:24px}.forbidden-card{width:420px;padding:52px;text-align:center;background:#fff;border-radius:10px;box-shadow:0 8px 30px rgba(32,67,82,.1)}.lock{width:76px;height:76px;line-height:76px;margin:0 auto 22px;border-radius:50%;background:#eef4fb;color:#409eff;font-size:34px}.forbidden-card h2{margin:0 0 28px;color:#314f60}
</style>
