import {Event} from "./Event"

export default class extends Event {
  labels = []
  color = 'purple'
  app = 'profiler'

  get type() {
    return 'profiler'
  }

  get route() {
    return {
      index: `/${this.app}`,
      show: `/${this.app}/${this.id}`,
      json: `/api/event/${this.id}`,
    }
  }

  get edges() {
    return this.event.edges
  }

  get peaks() {
    return this.event.peaks
  }

  get origin() {
    return {
      tags: this.event.tags,
      app: this.event.app_name,
      hostname: this.event.hostname
    }
  }
}
