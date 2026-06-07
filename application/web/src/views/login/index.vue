<template>
  <div class="login-container">
    <el-form class="login-form">
      <div class="title-container">
        <h3>基于区块链的食用油运输监管系统</h3>
        <p>全流程可信存证 · 运输轨迹监管 · 产品一键追溯</p>
      </div>
      <template v-if="mode === 'login'">
        <el-input v-model="loginForm.username" prefix-icon="el-icon-user" placeholder="请输入账号">
          <el-dropdown slot="suffix" trigger="click" @command="useDemo">
            <i class="el-icon-arrow-down account-arrow" title="选择演示账号" />
            <el-dropdown-menu slot="dropdown">
              <el-dropdown-item v-for="item in demoAccounts" :key="item.username" :command="item.username">
                <span class="account-role">{{ item.role }}</span>
                <b>{{ item.username }}</b>
              </el-dropdown-item>
            </el-dropdown-menu>
          </el-dropdown>
        </el-input>
        <el-input v-model="loginForm.password" prefix-icon="el-icon-lock" type="password" placeholder="请输入密码" @keyup.enter.native="handleLogin" />
        <el-button type="primary" :loading="loading" @click="handleLogin">登 录</el-button>
        <el-button class="text-button" type="text" @click="mode='register'">参与主体注册申请</el-button>
      </template>
      <template v-else>
        <div class="grid">
          <el-input v-model="registerForm.username" placeholder="登录账号" />
          <el-input v-model="registerForm.displayName" placeholder="负责人姓名" />
          <el-input v-model="registerForm.password" type="password" placeholder="登录密码（至少6位）" />
          <el-input v-model="registerForm.phone" placeholder="联系电话" />
        </div>
        <el-input v-model="registerForm.organization" placeholder="单位或组织名称" />
        <el-select v-model="registerForm.role" placeholder="申请角色">
          <el-option v-for="r in roles" :key="r" :label="r" :value="r" />
        </el-select>
        <el-button type="primary" :loading="loading" @click="handleRegister">提交注册申请</el-button>
        <el-button class="text-button" type="text" @click="mode='login'">返回登录</el-button>
        <div class="demo-tip">注册后需由系统管理员完成主体资质审核，审核通过后方可登录。</div>
      </template>
    </el-form>
  </div>
</template>

<script>
export default {
  name: 'Login',
  data() {
    return {
      mode: 'login', loading: false,
      roles: ['原料供应商', '榨油厂', '运输人员', '零售商', '监管机构'],
      loginForm: { username: 'admin', password: '123456' },
      registerForm: { username: '', password: '', displayName: '', role: '', organization: '', phone: '' },
      demoAccounts: [
        { role: '系统管理员', username: 'admin' },
        { role: '原料供应商', username: 'supplier' },
        { role: '榨油厂', username: 'factory' },
        { role: '运输人员', username: 'driver' },
        { role: '零售商', username: 'retailer' },
        { role: '监管机构', username: 'regulator' }
      ]
    }
  },
  methods: {
    handleLogin() {
      this.loading = true
      this.$store.dispatch('user/login', this.loginForm).then(() => this.$router.push('/')).finally(() => { this.loading = false })
    },
    handleRegister() {
      this.loading = true
      this.$store.dispatch('user/register', this.registerForm).then(res => { this.$message.success(res.message); this.mode = 'login' }).finally(() => { this.loading = false })
    },
    useDemo(username) {
      this.loginForm = { username, password: '123456' }
    }
  }
}
</script>

<style lang="scss" scoped>
.login-container{min-height:100%;background-image:url("../../assets/login_images/nature.jpg");background-position:center;background-size:cover;background-repeat:no-repeat;display:flex;align-items:center;justify-content:center}
.login-form{width:560px;padding:38px 46px;background:#fff;border:1px solid rgba(42,91,84,.15);border-radius:14px;box-shadow:0 20px 60px rgba(20,61,57,.2)}
.title-container{text-align:center;color:#254a48;margin-bottom:28px}.title-container h3{font-size:25px;margin:0 0 10px}.title-container p{font-size:13px;color:#6b8f8b;letter-spacing:2px}
.el-input,.el-select{width:100%;margin-bottom:16px}.el-button--primary{width:100%;margin:4px 0 0}.text-button{width:100%;margin:8px 0 0!important;color:#b7d9dc}.grid{display:grid;grid-template-columns:1fr 1fr;gap:0 12px}.demo-tip{margin-top:18px;color:#b7c5cf;font-size:12px;line-height:1.7;text-align:center}
.text-button{color:#477d78}.account-arrow{height:40px;line-height:40px;padding:0 8px;color:#527c78;cursor:pointer}
</style>
<style>
.el-dropdown-menu__item .account-role{display:inline-block;width:82px;color:#718985;margin-right:12px}
</style>
