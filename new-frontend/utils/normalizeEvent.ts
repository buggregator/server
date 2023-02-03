import { NormalizedEvent, ServerEvent, Monolog, SMTP, Sentry, VarDump } from "~/config/types";
import { EVENT_TYPES } from "~/config/constants";

const normalizeObjectValue = (object: object | unknown[]): object =>
  Object.entries(object).reduce((acc: object, [key, value]) => ({
    ...acc,
    [key]: value
  }), {})

export const normalizeFallbackEvent = (event: ServerEvent<unknown>): NormalizedEvent => ({
  id: event.uuid,
  type: 'unknown',
  labels: [event.type],
  origin: null,
  serverName: "",
  date: new Date(event.timestamp * 1000),
  payload: event.payload
})

// TODO: need to update normalize fn
export const normalizeInspectorEvent = normalizeFallbackEvent
// TODO: need to update normalize fn
export const normalizeProfilerEvent = normalizeFallbackEvent

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


export const normalizeSentryEvent = (event: ServerEvent<Sentry>): NormalizedEvent => ({
    id: event.uuid,
    type: EVENT_TYPES.SENTRY,
    labels: [EVENT_TYPES.SENTRY, 'exception'],
    origin: {
      logger: event.payload.logger,
      environment: event.payload.environment
    },
    serverName: event.payload.server_name,
    date: new Date(event.timestamp * 1000),
    payload: event.payload
  })

export const normalizeSMTPEvent = (event: ServerEvent<SMTP>): NormalizedEvent => ({
  id: event.uuid,
  type: EVENT_TYPES.SMTP,
  labels: [EVENT_TYPES.SMTP],
  origin: null,
  serverName: "",
  date: new Date(event.timestamp * 1000),
  payload: event.payload
})

export const normalizeVarDumpEvent = (event: ServerEvent<VarDump>): NormalizedEvent => ({
  id: event.uuid,
  type: EVENT_TYPES.VAR_DUMP,
  labels: [EVENT_TYPES.VAR_DUMP],
  origin: {
    file: event.payload.context.source.file,
    name: event.payload.context.source.name,
    line_number: event.payload.context.source.line,
  },
  serverName: "",
  date: new Date(event.timestamp * 1000),
  payload: event.payload
})
