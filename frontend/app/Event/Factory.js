import RayEvent, {EventHandler as RayEventHandler} from "./Ray";
import SentryEvent from "./Sentry";
import MonologEvent from "./Monolog";
import SmtpEvent from "./Smtp";
import VarDumpEvent from "./VarDump";
import InspectorEvent from "./Inspector";
import ProfilerEvent from "./Profiler";

const eventTypes = {
  ray: ({json, store}) => {
    const event = new RayEvent(json.payload, json.uuid, json.timestamp);
    if (new RayEventHandler({event, store}).handle()) {
      return event
    }
  },
  sentry: ({json}) => new SentryEvent(json.payload, json.uuid, json.timestamp),
  monolog: ({json}) => new MonologEvent(json.payload, json.uuid, json.timestamp),
  smtp: ({json}) => new SmtpEvent(json.payload, json.uuid, json.timestamp),
  inspector: ({json}) => new InspectorEvent(json.payload, json.uuid, json.timestamp),
  profiler: ({json}) => new ProfilerEvent(json.payload, json.uuid, json.timestamp),
  'var-dump': ({json}) => new VarDumpEvent(json.payload, json.uuid, json.timestamp)
}

export default {
  subscribed: false,

  init({store, channel}) {
    if (this.subscribed) {
      return;
    }

    channel.listen(`event.received`, data => {

      const event = this.create({json: data, store})
      if (!event) {
        return;
      }

      store.commit('events/push', event)

    }).listen(`event.deleted`, e => {

      store.commit('events/delete', e.uuid)

    }).listen(`event.cleared`, e => {
      store.commit('events/clear', e.type)
    })

    this.subscribed = true
  },

  create({json, store}) {
    const type = json.type.toLowerCase()

    if (eventTypes.hasOwnProperty(type)) {
      return eventTypes[type]({json, store})
    }

    throw new Error(`Event type [${type}] is not found.`)
  }
}
