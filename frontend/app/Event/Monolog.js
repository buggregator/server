import {Event} from "./Event"

const colorMap = {
  CRITICAL: 'red',
  ERROR: 'red',
  ALERT: 'red',
  EMERGENCY: 'red',
  WARNING: 'orange',
  INFO: 'blue',
  NOTICE: 'blue',
  DEBUG: 'gray'
}

export default class extends Event {
  labels = ['Monolog']
  app = 'monolog'

  constructor(event, id, timestamp) {
    super(event, id, timestamp)

    this.color = colorMap[this.level] || 'gray'
    this.labels.push(this.level)
  }

  get level() {
    return this.event.level_name || 'DEBUG'
  }

  get type() {
    return 'monolog'
  }

  get text() {
    return this.event.message || ''
  }

  get payloads() {
    return this.event.context || []
  }

  get fields() {
    return this.event.extra || []
  }

  get serverName() {
    return this.event.channel || null
  }
}
