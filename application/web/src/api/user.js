import request from '@/utils/request'

export const login = data => request({ url: '/login', method: 'post', data })
export const register = data => request({ url: '/register', method: 'post', data })
export const getInfo = () => request({ url: '/info', method: 'get' })
export const logout = () => request({ url: '/logout', method: 'post' })
export const getUsers = () => request({ url: '/admin/users', method: 'get' })
export const updateUserStatus = data => request({ url: '/admin/users/status', method: 'post', data })
export const getLogs = kind => request({ url: '/admin/logs', method: 'get', params: { kind }})
