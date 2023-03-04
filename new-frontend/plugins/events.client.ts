import { apiTransport } from '~/utils/events-transport'
import { useEventStore } from "~/stores/events";
import { EventId, OneOfValues, ServerEvent } from "~/config/types";
import { EVENT_TYPES } from "~/config/constants";
import { storeToRefs } from "pinia";


export default defineNuxtPlugin(() => {
  const eventsStore = useEventStore();
  const onEventReceiveCb = (event: ServerEvent<unknown>) => {
    eventsStore.addEvents([event]);
  }

  const {
    deleteEvent,
    deleteEventsAll,
    deleteEventsByType,
  } = apiTransport({
    onEventReceiveCb,
  });

  const removeAll = () => {
    deleteEventsAll();
    eventsStore.removeEvents()
  }

  const removeById = (eventId: EventId) => {
    deleteEvent(eventId);
    eventsStore.removeEventById(eventId);
  }

  const removeByType = (type: OneOfValues<typeof EVENT_TYPES>) => {
    deleteEventsByType(type);
    eventsStore.removeEventsByType(type);
  }

  const { getAvailableEvents, getEventsByType } = eventsStore
  const { events } = storeToRefs(eventsStore)

  return {
    provide: {
      events: {
        items: events,
        getItemsByType: getEventsByType,
        getAll: getAvailableEvents,
        removeAll,
        removeByType,
        removeById,
      }
    }
  }
})
