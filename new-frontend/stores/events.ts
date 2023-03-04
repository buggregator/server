import { defineStore } from 'pinia';
import { EventId, OneOfValues, ServerEvent } from "~/config/types";
import { EVENT_TYPES } from "~/config/constants";

export const useEventStore = defineStore('useEventStore', {
  state: () => ({
    events: [] as ServerEvent<unknown>[],
    eventsLoading: false
  }),
  getters: {
    eventIdList: (state) => state.events.map(({ uuid }) => uuid),
  },
  actions: {
    getAvailableEvents() {
      this.eventsLoading = true;

      fetch(`https://test.buggregator.dev/api/events`)
        .then((response) => response.json())
        .then((response) => {
          if (response?.data?.length > 0) {
            return response.data
          }

          throw new Error('Fetch Error')
        })
        .then((events: ServerEvent<unknown>[]) => {
          this.events = events.concat(this.events);

          this.eventsLoading = false
        })
    },
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
  },
})
