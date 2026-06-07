const getters = {
  sidebar: state => state.app.sidebar,
  device: state => state.app.device,
  token: state => state.user.token,
  avatar: state => state.user.avatar,
  name: state => state.user.name,
  username: state => state.user.username,
  role: state => state.user.role,
  userType: state => state.user.role,
  organization: state => state.user.organization
}
export default getters
