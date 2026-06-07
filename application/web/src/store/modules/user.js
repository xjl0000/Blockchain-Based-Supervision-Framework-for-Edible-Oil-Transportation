import { login, logout, getInfo, register } from '@/api/user'
import { getToken, setToken, removeToken } from '@/utils/auth'
import { resetRouter } from '@/router'

const getDefaultState = () => ({ token: getToken(), name: '', username: '', role: '', organization: '', avatar: '' })
const state = getDefaultState()
const mutations = {
  RESET_STATE: state => Object.assign(state, getDefaultState()),
  SET_TOKEN: (state, token) => { state.token = token },
  SET_USER: (state, user) => Object.assign(state, user)
}
const actions = {
  login({ commit }, form) {
    return login(form).then(res => { commit('SET_TOKEN', res.jwt); setToken(res.jwt); return res })
  },
  register(_, form) { return register(form) },
  getInfo({ commit }) {
    return getInfo().then(res => {
      commit('SET_USER', { name: res.name, username: res.username, role: res.role, organization: res.organization })
      return res
    })
  },
  logout({ commit }) {
    return logout().finally(() => { removeToken(); resetRouter(); commit('RESET_STATE') })
  },
  resetToken({ commit }) { removeToken(); commit('RESET_STATE'); return Promise.resolve() }
}
export default { namespaced: true, state, mutations, actions }
