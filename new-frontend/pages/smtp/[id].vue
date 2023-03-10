<template>
  <main class="sentry-event">
    <PageSmtp
v-if="event"
              :event="event"
              :html-source="html"
              @delete="clearEvent"
    />
  </main>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import { EventId, SMTP, ServerEvent } from "~/config/types";
import { useNuxtApp, useRoute } from "#app";
import { normalizeSMTPEvent } from "~/utils/normalize-event";
import PageSmtp from "~/components/PageSmtp/PageSmtp.vue";
import {REST_API_URL} from "~/utils/events-transport";

export default defineComponent({
  components: { PageSmtp },
  setup() {
    const route = useRoute();
    const eventId = route.params.id as EventId;

    if (process.client) {
      const { $events } = useNuxtApp();
      const serverEvent = $events.getItemById(
        eventId
      ) as ServerEvent<SMTP> | null;

      return {
        event: serverEvent ? normalizeSMTPEvent(serverEvent) : null,
        html: `<iframe src="${REST_API_URL}/api/smtp/${eventId}/html"/>`,
        clearEvent: () => $events.removeById(eventId),
      };
    }

    return {
      event: null,
      html: '',
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
