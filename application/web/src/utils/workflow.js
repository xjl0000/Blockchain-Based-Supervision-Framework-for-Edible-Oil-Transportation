export const names = { raw_draft: '原料草稿', pending_factory: '待榨油厂接收', returned_supplier: '已退回供应商', factory_received: '原料已接收', processed: '加工完成', pending_transport: '待运输人员接收', transport_accepted: '运输任务已接收', in_transit: '运输中', pending_retail: '待零售商收货', completed: '全流程已完成', pending_accept: '待接收', accepted: '已接收' }
export const statusName = value => names[value] || value
export const tagType = value => value === 'completed' ? 'success' : value.indexOf('returned') >= 0 ? 'danger' : value.indexOf('pending') >= 0 ? 'warning' : 'primary'
