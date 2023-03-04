import { apiConnection } from '~/utils/events-transport'
import { useEventStore } from "~/stores/events";
import { EventId, OneOfValues, ServerEvent } from "~/config/types";
import { EVENT_TYPES } from "~/config/constants";


export default defineNuxtPlugin(() => {
  const eventsStore = useEventStore();
  const onEventReceiveCb = (event: ServerEvent<unknown>) => {
    eventsStore.addEvents([event]);
  }

  const {
    deleteEvent,
    deleteEventsAll,
    deleteEventsByType,
  } = apiConnection({
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

  return {
    provide: {
      events: {
        removeAll,
        removeByType,
        removeById,
      }
    }
  }
})
