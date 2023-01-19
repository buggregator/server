import { EVENT_TYPES } from "~/config/constants";

export type OneOfValues<T> = T[keyof T];
export type EventId = string;
export type StatusCode = string; // TODO: update type
export type UnixTimeStamp = number; // TODO: update type

export interface ServerEvent {
  uuid: EventId,
  type: string,
  payload: Object,
  project_id: string|null,
  timestamp: UnixTimeStamp
}


export interface NormalizedEvent {
  id: EventId,
  type: OneOfValues<typeof EVENT_TYPES>,
  labels: string[],
  origin: Object,
  serverName: string,
  date: Date,
  payload: unknown
}

export interface MonologEvent extends ServerEvent {
  payload: {
    message: string,
    context: unknown,
    level: StatusCode,
    level_name: string,
    channel: string,
    datetime: Date,
    extra: unknown[],
  },
}

