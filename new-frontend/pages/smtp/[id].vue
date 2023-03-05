<template>
  <main class="sentry-event">
    <EventSmtp v-if="event" :event="event" />
  </main>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import { EventId, Sentry, ServerEvent } from "~/config/types";
import { useNuxtApp, useRoute } from "#app";
import { normalizeSMTPEvent } from "~/utils/normalize-event";
import EventSmtp from "~/components/EventSmtp/EventSmtp.vue";

export default defineComponent({
  components: { EventSmtp },
  setup() {
    const route = useRoute();
    const eventId = route.params.id as EventId;

    if (process.client) {
      const { $events } = useNuxtApp();
      const serverEvent = $events.getItemById(
        eventId
      ) as ServerEvent<Sentry> | null;

      return {
        event: serverEvent ? normalizeSMTPEvent(serverEvent) : null,
        clearEvent: () => $events.removeById(eventId),
      };
    }

    return {
      event: null,
      clearEvent: () => {},
    };
  },
  head() {
    const route = useRoute();

    return {
      title: `SMTP > ${route.params.id} | Buggregator`,
    };
  },
});
</script>

<style lang="scss" scoped>
.sentry-event {
  @apply h-full w-full;
}
</style>
