import axios from 'axios'
import { Message } from 'element-ui'
import store from '@/store'
import { getToken } from '@/utils/auth'

const service = axios.create({ baseURL: process.env.VUE_APP_BASE_API, timeout: 15000 })
const rfc3339 = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})$/

function normalizeDates(value) {
  if (Array.isArray(value)) return value.map(normalizeDates)
  if (value && typeof value === 'object') {
    Object.keys(value).forEach(key => { value[key] = normalizeDates(value[key]) })
    return value
  }
  if (typeof value === 'string' && rfc3339.test(value)) {
    return value.slice(0, 19).replace('T', ' ')
  }
  return value
}

service.interceptors.request.use(config => {
  if (store.getters.token) config.headers.Authorization = `Bearer ${getToken()}`
  return config
})

service.interceptors.response.use(response => normalizeDates(response.data), error => {
  const message = error.response && error.response.data ? error.response.data.message : error.message
  Message({ message, type: 'error', duration: 4000 })
  return Promise.reject(error)
})

export default service
