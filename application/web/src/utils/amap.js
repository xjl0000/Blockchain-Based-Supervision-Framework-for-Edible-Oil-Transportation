import AMapLoader from '@amap/amap-jsapi-loader'

let loaderPromise

export function loadAMap() {
  if (!process.env.VUE_APP_AMAP_KEY || !process.env.VUE_APP_AMAP_SECURITY_CODE) {
    return Promise.reject(new Error('高德地图 Key 或安全密钥未配置'))
  }

  if (!loaderPromise) {
    loaderPromise = AMapLoader.load({
      key: process.env.VUE_APP_AMAP_KEY,
      securityJsCode: process.env.VUE_APP_AMAP_SECURITY_CODE,
      version: '2.0',
      plugins: ['AMap.Scale', 'AMap.ToolBar']
    })
  }

  return loaderPromise
}
