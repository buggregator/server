import sentryEventMock from '~/mocks/sentry.json'
import monologEventMock from '~/mocks/monolog.json'
import smtpEventMock from '~/mocks/smtp.json'
import profilerEventMock from '~/mocks/profiler.json'
import { defineStore } from 'pinia';
import { EventId, OneOfValues } from "~/config/types";
import { EVENT_TYPES } from "~/config/constants";


const getEvents = () => {
  // TODO: api call
  const fakeEvents = [
    sentryEventMock,
    monologEventMock,
    smtpEventMock,
    profilerEventMock
  ]

  return {
    fakeEvents,
  }
}

/* eslint-disable import/prefer-default-export */
export const useEventStore = defineStore('useEventStore', {
  state: () => getEvents(),
  getters: {
    events: (state) => state.fakeEvents,
    eventIdList: (state) => state.fakeEvents.map(({ uuid }) => uuid),
  },
  actions: {
    removeEventByUuid(eventUuid: EventId) {
      this.fakeEvents = this.events.filter(({ uuid }) => uuid !== eventUuid)
    },

    removeEventsType(eventType: OneOfValues<typeof EVENT_TYPES>) {
      this.fakeEvents = this.fakeEvents.filter(({ type }) => type !== eventType)
    },
    removeAllEvents() {
      this.fakeEvents = []
    },
  },
})
