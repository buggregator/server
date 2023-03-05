import {Centrifuge} from 'centrifuge'
import { EventId, OneOfValues, ServerEvent } from "~/config/types";
import { EVENT_TYPES } from "~/config/constants";

const WS_URL = `wss://test.buggregator.dev/connection/websocket`// ||`${wsProtocol}://${host}/connection/websocket`

export type LoggerParams = [string, unknown]
export interface ApiConnection {
  onEventReceiveCb: (param: ServerEvent<unknown>) => void
  loggerCb?: (params: LoggerParams) => void
}

const defaultLogger = (params: LoggerParams) => {
  console.info(`[ApiConnection logger]:Centrifuge "${params[0]}" called with params: "${JSON.stringify(params[1])}"`)
}

export const apiTransport = ({
    onEventReceiveCb,
    loggerCb = defaultLogger,
  }: ApiConnection) => {
  const centrifuge = new Centrifuge(WS_URL)


  const eventsChannel = centrifuge.newSubscription('events')

  centrifuge.on('connected', (ctx) => {
    loggerCb(['connected', ctx]);
  });

  eventsChannel.on('publication', (ctx) => {
    loggerCb(['publication', ctx]);
    const event = ctx?.data?.data || null

    if (event) {
      onEventReceiveCb(event)
    }
  });

  centrifuge.on('disconnected', (ctx) => {
    loggerCb(['disconnected', ctx]);
  });

  eventsChannel.subscribe();
  centrifuge.connect();

  const deleteEvent = (eventId: EventId) => {
    centrifuge.rpc(`delete:api/events/${eventId}`, undefined)
  }

  const deleteEventsAll = () => {
    centrifuge.rpc(`delete:api/events`, undefined)
  }

  const deleteEventsByType = (type: OneOfValues<typeof EVENT_TYPES>) => {
    centrifuge.rpc(`delete:api/events`, {type})
  }

  const getEventsAll = fetch(`https://test.buggregator.dev/api/events`)
    .then((response) => response.json())
    .then((response) => {
      if (response?.data?.length > 0) {
        return response.data
      }

      throw new Error('Fetch Error')
    })
    .then((events: ServerEvent<unknown>[]) => events)


  return {
    getEventsAll,
    deleteEvent,
    deleteEventsAll,
    deleteEventsByType,
  }
}
