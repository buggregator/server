import { defineStore } from 'pinia';
import { EventId, OneOfValues, ServerEvent } from "~/config/types";
import { EVENT_TYPES } from "~/config/constants";


export const useEventStore = defineStore('useEventStore', {
  state: () => ({
    events: [] as ServerEvent<unknown>[],
    eventsLoading: false
  }),
  getters: {
    sentryEvents: (state) => state.events.filter(({ type }) => type === EVENT_TYPES.SENTRY),
    inspectorEvents: (state) => state.events.filter(({ type }) => type === EVENT_TYPES.INSPECTOR),
    profilerEvents: (state) => state.events.filter(({ type }) => type === EVENT_TYPES.PROFILER),
    smtpEvents: (state) => state.events.filter(({ type }) => type === EVENT_TYPES.SMTP),
    httpDumpEvents: (state) => state.events.filter(({ type }) => type === EVENT_TYPES.HTTP_DUMP)
  },
  actions: {
    removeEventById(eventUuid: EventId) {
      this.events = this.events.filter(({ uuid }) => uuid !== eventUuid)
    },
    removeEvents() {
      this.events.length = 0
    },
    removeEventsByType(eventType: OneOfValues<typeof EVENT_TYPES>) {
      this.events = this.events.filter(({ type }) => type !== eventType);
    },
    addEvents(events: ServerEvent<unknown>[]) {
      events.forEach((event) => {
        const isExistedEvent = this.events.some((el) => el.uuid === event.uuid)
        if (!isExistedEvent) {
          this.events.unshift(event)
        }
      })
    },
    getEventById(id: EventId): ServerEvent<unknown> | null {
      return this.events.find(({ uuid }) => String(uuid) === String(id)) || null
    }
  },
})
