import {Event} from "./Event"

export default class extends Event {
  labels = []
  color = 'orange'
  app = 'smtp'

  get subject() {
    return this.event.subject
  }

  get type() {
    return 'Smtp'
  }
}
