import { defineStore } from 'pinia';
import { EventId, OneOfValues, ServerEvent } from "~/config/types";
import { EVENT_TYPES } from "~/config/constants";

export const useEventStore = defineStore('useEventStore', {
  state: () => ({
    events: [] as ServerEvent<unknown>[],
    eventsLoading: false
  }),
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
        this.events.unshift(event)
      })
    },
    getEventsByType(type: OneOfValues<typeof EVENT_TYPES>) {
      return this.events.filter((event) => event.type === type)
    }
  },
})
