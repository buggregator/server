import {Event} from "./Event"

export default class extends Event {
  labels = ['smtp']
  color = 'orange'
  app = 'smtp'

  get subject() {
    return this.event.subject
  }

  get type() {
    return 'Smtp'
  }
}
