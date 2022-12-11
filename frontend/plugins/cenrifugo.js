import {WsClient} from '~/Websocket/Client'
import {Centrifuge} from 'centrifuge'
import Factory from "../app/Event/Factory"

const host = window.location.host
const wsProtocol = window.location.protocol === 'https:' ? 'wss' : 'ws'

const WS_URL = process.env.WS_URL || `${wsProtocol}://${host}/connection/websocket`

const subscribe = (client, store) => client.connect()
  .then(ctx => {
    store.commit('ws/connect')
  })
  .catch(err => {
    store.commit('ws/disconnect')
  })
  .then(ctx => {
    const channel = client.eventsChannel()
    Factory.init({channel, store})
  })

export default async (ctx, inject) => {
  const centrifuge = new Centrifuge(WS_URL)

  const client = new WsClient(centrifuge, ctx.$logger.withPrefix('WS'))
  await subscribe(client, ctx.store)

  inject('ws', client)
  ctx.$ws = client
}

