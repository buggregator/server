import {Event} from "./Event"

const labelsMap = {
  create_lock: 'Pause',
  trace: 'Trace',
  table: 'Table',
  caller: 'Caller',
  measure: 'Measure',
  event: 'Event',
  job_event: 'Job',
  json_string: 'Json',
  cache: 'Cache',
  view: 'View',
  eloquent_model: 'Eloquent model',
  executed_query: 'Query',
  application_log: 'Monolog',
  mailable: 'Mail',
  log: 'Log',
  custom: null
}

const handlers = {
  clear_all: ({event, store}) => {
    store.commit('clearEvents')
    return false
  },
  new_screen: ({event, store}) => {
    store.commit('switchScreen', event.content('name'))
    return false
  },
  remove: ({event, store}) => {
    store.commit('deleteEvent', event.uuid)
    return false
  },
  hide: ({event, store}) => {
    store.commit('toggleEventState', [event.uuid, true])
    return false
  },
  show: ({event, store}) => {
    store.commit('toggleEventState', [event.uuid, false])
    return false
  },
  notify: ({event}) => {
    notify({
      title: "Hello from Ray",
      text: event.payloads[0].content.value,
      duration: -1
    })
    return false
  }
}

export default class extends Event {
  app = 'ray'

  constructor(event, id, timestamp) {
    super(event, id, timestamp)

    this.labels = this.collectLabels()
    this.color = this.detectColor()
  }

  get uuid() {
    return this.event.uuid
  }

  get type() {
    return this.payloads[0].type
  }

  get serverName() {
    return this.payloads[0].origin.hostname
  }

  get payloads() {
    return this.event.payloads || []
  }

  content(field) {
    return this.payloads[0].content[field]
  }

  detectColor() {
    let color = this.color
    this.payloads.forEach(function (payload) {
      if (payload.content.color) {
        color = payload.content.color
      }
    })

    return color
  }

  collectLabels() {
    let labels = [];

    this.payloads.forEach(function (payload) {
      if (payload.content.label) {
        const label = labelsMap.hasOwnProperty(payload.content.label)
          ? labelsMap[payload.content.label]
          : payload.content.label;

        if (label && !labels.includes(label)) {
          labels.push(label)
        }
      }

      const typeLabel = labelsMap.hasOwnProperty(payload.type)
        ? labelsMap[payload.type]
        : payload.type;

      if (typeLabel && !labels.includes(typeLabel)) {
        labels.push(typeLabel)
      }
    })

    return labels
  }

  merge(event) {
    this.event = _.merge(event, this.event)
    this.labels = this.collectLabels()
    this.color = this.detectColor()
  }
}

export class EventHandler {
  constructor(ctx) {
    this.ctx = ctx
  }

  handle() {
    if (handlers.hasOwnProperty(this.ctx.event.type)) {
      return handlers[this.ctx.event.type](this.ctx)
    }

    return true
  }
}
