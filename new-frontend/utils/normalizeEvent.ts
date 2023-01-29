import { NormalizedEvent, ServerEvent, Monolog } from "~/config/types";
import { EVENT_TYPES } from "~/config/constants";

const normalizeObjectValue = (object: Object | unknown[]): Object =>
  Object.entries(object).reduce((acc: Object, [key, value]) => ({
    ...acc,
    [key]: value
  }), {})

export const normalizeInspectorEvent = () => {}

export const normalizeMonologEvent = (event: ServerEvent<Monolog>): NormalizedEvent => ({
  id: event.uuid,
  type: EVENT_TYPES.MONOLOG,
  labels: [EVENT_TYPES.MONOLOG, event.payload.level_name],
  origin: null,
  serverName: event.payload.channel,
  date: new Date(event.timestamp * 1000),
  payload: {
    ...event.payload,
    context: normalizeObjectValue(event.payload.context),
    extra: normalizeObjectValue(event.payload.extra)
  }
})

export const normalizeProfilerEvent = () => {}
export const normalizeSentryEvent = () => {}
export const normalizeSMTPEvent = (event: ServerEvent<SMTP>): NormalizedEvent => ({
  id: event.uuid,
  type: EVENT_TYPES.SMTP,
  labels: [EVENT_TYPES.SMTP],
  origin: null,
  serverName: "",
  date: new Date(event.timestamp * 1000),
  payload: event.payload
})

export const normalizeVarDumpEvent = () => {}
