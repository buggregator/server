import {Centrifuge} from 'centrifuge'
import {EventId, OneOfValues, ServerEvent} from "~/config/types";
import {EVENT_TYPES} from "~/config/constants";

// A developer not always has a possibility to configure ENV variables,
// so we need to guess Api and WS connection urls.
const guessWsConnection = (): string => {
  const WS_HOST = window.location.host
  const WS_PROTOCOL = window.location.protocol === 'https:' ? 'wss' : 'ws'

  return `${WS_PROTOCOL}://${WS_HOST}/connection/websocket`;
}

const guessRestApiConnection = (): string => {
  const API_HOST = window.location.host
  const API_PROTOCOL = window.location.protocol === 'https:' ? 'https' : 'http'

  return `${API_PROTOCOL}://${API_HOST}`;
}

export const REST_API_URL = (import.meta.env.VITE_EVENTS_REST_API as string) || guessRestApiConnection()
export const WS_URL = (import.meta.env.VITE_EVENTS_WS_API as string) || guessWsConnection()

export type LoggerParams = [string, unknown]

export interface ApiConnection {
  onEventReceiveCb: (param: ServerEvent<unknown>) => void
  loggerCb?: (params: LoggerParams) => void
}

const defaultLogger = (params: LoggerParams) => {
  console.info(`[ApiConnection logger]:Centrifuge "${params[0]}" called with params: "${JSON.stringify(params[1])}"`)
}

export const apiTransport = ({ onEventReceiveCb, loggerCb = defaultLogger, }: ApiConnection) => {
  const centrifuge = new Centrifuge(WS_URL)

  centrifuge.on('connected', (ctx) => {
    loggerCb(['connected', ctx]);
  });

  centrifuge.on('publication', (ctx) => {
    loggerCb(['publication', ctx]);

    // We need to handle only events from the channel 'events' with event name 'event.received'
    if (ctx.channel === 'events' && ctx.data?.event === 'event.received') {
      const event = ctx?.data?.data || null
      onEventReceiveCb(event)
    }
  });

  centrifuge.on('disconnected', (ctx) => {
    loggerCb(['disconnected', ctx]);
  });

  centrifuge.connect();

  const deleteEvent = (eventId: EventId) => {
    centrifuge.rpc(`delete:api/event/${eventId}`, undefined)
  }

  const deleteEventsAll = () => {
    centrifuge.rpc(`delete:api/events`, undefined)
  }

  const deleteEventsByType = (type: OneOfValues<typeof EVENT_TYPES>) => {
    centrifuge.rpc(`delete:api/events`, {type})
  }

  const getEventsAll = fetch(`${REST_API_URL}/api/events`)
    .then((response) => response.json())
    .then((response) => {
      if (response?.data?.length > 0) {
        return response.data
      }

      throw new Error('Fetch Error')
    })
    .then((events: ServerEvent<unknown>[]) => events)

  const getEvent = (id: EventID) => fetch(`${REST_API_URL}/api/event/${id}`)
    .then((response) => response.json())
    .then((response) => {
      if (response?.data) {
        return response.data as ServerEvent<unknown>[]
      }
      return null
    })


  return {
    getEventsAll,
    getEvent,
    deleteEvent,
    deleteEventsAll,
    deleteEventsByType,
    makeEventUrl: (id) => `${REST_API_URL}/api/event/${id}`
  }
}
