import { NormalizedEvent, MonologEvent } from "~/config/types";
import { EVENT_TYPES } from "~/config/constants";

export const normalizeInspectorEvent = () => {}

export const normalizeMonologEvent: NormalizedEvent = (event: MonologEvent) => ({
    id: event.uuid,
    type: EVENT_TYPES.MONOLOG,
    labels: [EVENT_TYPES.MONOLOG, event.payload.level_name],
    origin: null,
    serverName: event.payload.channel,
    date: '',
    payload: event.payload
  })

export const normalizeProfilerEvent = () => {}
export const normalizeSentryEvent = () => {}
export const normalizeSMTPEvent = () => {}
export const normalizeVarDumpEvent = () => {}
