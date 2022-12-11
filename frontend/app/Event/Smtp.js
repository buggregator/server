import {Event} from "./Event"

export default class extends Event {
  labels = ['smtp']
  color = 'blue'
  app = 'smtp'

  get subject() {
    return this.event.subject
  }

  get serverName() {
    return this.event.server_name
  }

  get type() {
    return 'Smtp'
  }
}
