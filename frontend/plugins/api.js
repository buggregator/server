import apiMethods from '~/api'

export default function (ctx, inject) {
  const api = new class Api {
    get events() {
      return {
        list: apiMethods.showEvents(ctx),
        show: apiMethods.showEvent(ctx),
        clear: apiMethods.clearEvents(ctx),
        delete: apiMethods.deleteEvent(ctx),
      }
    }
  }

  inject('api', api)
  ctx.$api = api
}
