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

export interface Sentry {
  event_id: string,
  timestamp: number,
  platform: string,
  sdk: {
    name: string,
    version: string,
  },
  logger: string,
  server_name: string,
  environment: string,
  modules: object,
  extra: unknown,
  tags: object,
  contexts: object,
  exception: {
    values: {
      stacktrace: {
        frames: unknown[]
      },
      [key: string]: unknown
    }[]
  }
}

export interface VarDump {
  payload: {
    type: string,
    value: string
  },
  context: {
    timestamp: number,
    cli: {
      command_line: string,
      identifier: string
    },
    source: {
      name: string,
      file: string,
      line: number,
      file_excerpt: boolean
    }
  }
}

export interface ProfilerCost {
  [key: string]: number,
  "ct": number,
  "wt": number,
  "cpu": number,
  "mu": number,
  "pmu": number
}

export interface ProfilerEdge {
  caller: unknown,
  callee: unknown,
  cost: ProfilerCost
}

export interface Profiler {
  tags: {
    [key: string]: string | null | number
  },
  app_name: string,
  hostname: string,
  date: number,
  peaks: ProfilerCost,
  edges: {
    [key: string]: ProfilerEdge
  }
}


export interface ServerEvent<T> {
  uuid: EventId,
  type: OneOfValues<typeof EVENT_TYPES> | string,
  payload: T,
  project_id: string|null,
  timestamp: number
}

export interface NormalizedEvent {
  id: EventId,
  type: OneOfValues<typeof EVENT_TYPES> | string,
  labels: string[],
  origin: object | null,
  serverName: string,
  date: Date,
  payload: Monolog | SMTP | Sentry | VarDump | Profiler | unknown
}
