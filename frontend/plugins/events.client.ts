import {apiTransport} from '~/utils/events-transport'
import {useEventStore} from "~/stores/events";
import {EventId, OneOfValues, ServerEvent} from "~/config/types";
import {EVENT_TYPES} from "~/config/constants";
import {storeToRefs} from "pinia";

export default defineNuxtPlugin(() => {
  const eventsStore = useEventStore();

  const {
    deleteEvent,
    deleteEventsAll,
    deleteEventsByType,
    getEventsAll,
    makeEventUrl,
  } = apiTransport({
    onEventReceiveCb: (event: ServerEvent<unknown>) => {
      eventsStore.addEvents([event]);
    }
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

  const getAll = () => {
    getEventsAll.then((events: ServerEvent<unknown>[]) => {
      eventsStore.addEvents(events);
    })
  }

  const {events, varDumpEvents, sentryEvents, inspectorEvents, profilerEvents, smtpEvents, rayEvents, httpDumpEvents} = storeToRefs(eventsStore)

  const itemsGroupByType = {
    [EVENT_TYPES.SENTRY]: sentryEvents,
    [EVENT_TYPES.INSPECTOR]: inspectorEvents,
    [EVENT_TYPES.PROFILER]: profilerEvents,
    [EVENT_TYPES.SMTP]: smtpEvents,
    [EVENT_TYPES.RAY_DUMP]: rayEvents,
    [EVENT_TYPES.VAR_DUMP]: varDumpEvents,
    [EVENT_TYPES.HTTP_DUMP]: httpDumpEvents,
  }

  return {
    provide: {
      events: {
        items: events,
        itemsGroupByType,
        getItemById: eventsStore.getEventById,
        buildItemFetchUrl: makeEventUrl,
        getAll,
        removeAll,
        removeByType,
        removeById,
      }
    }
  }
})
