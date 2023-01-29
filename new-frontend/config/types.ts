import { EVENT_TYPES } from "~/config/constants";

export type OneOfValues<T> = T[keyof T];
export type EventId = string;
export type StatusCode = number; // TODO: update type
export type Email = string; // TODO: update type

type SMTPUser = {
  name: string;
  email: Email;
}

export interface Monolog {
  message: string,
  context: object,
  level: StatusCode,
  level_name: string,
  channel: string,
  datetime: string,
  extra: object,
}

export interface SMTP {
  id: string,
  from: SMTPUser[],
  reply_to: SMTPUser[],
  subject: string,
  to: SMTPUser[],
  cc: SMTPUser[],
  bcc: SMTPUser[],
  text: string,
  html: string,
  raw: string,
  attachments: unknown[]
}

export interface ServerEvent<T> {
  uuid: EventId,
  type: string,
  payload: T,
  project_id: string|null,
  timestamp: number
}

export interface NormalizedEvent {
  id: EventId,
  type: OneOfValues<typeof EVENT_TYPES>,
  labels: string[],
  origin: object | null,
  serverName: string,
  date: Date,
  payload: Monolog | SMTP | unknown
}

