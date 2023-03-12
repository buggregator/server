<template>
  <main class="sentry-event">
    <page-header button-title="Delete event" @delete="clearEvent">
      Sentry event: {{ event.id }}
    </page-header>

    <page-sentry v-if="event" :event="event" />
  </main>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import { EventId, Sentry, ServerEvent } from "~/config/types";
import { useNuxtApp, useRoute } from "#app";
import { normalizeSentryEvent } from "~/utils/normalize-event";
import PageSentry from "~/components/PageSentry/PageSentry.vue";
import PageHeader from "~/components/PageHeader/PageHeader.vue";

export default defineComponent({
  components: {
    PageSentry,
    PageHeader,
  },
  setup() {
    const route = useRoute();
    const eventId = route.params.id as EventId;

    if (process.client) {
      const { $events } = useNuxtApp();
      const serverEvent = $events.getItemById(
        eventId
      ) as ServerEvent<Sentry> | null;

      return {
        event: serverEvent ? normalizeSentryEvent(serverEvent) : null,
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
      title: `Sentry > ${route.params.id} | Buggregator`,
    };
  },
});
</script>

<style lang="scss" scoped>
.sentry-event {
  @apply h-full w-full;
}
</style>
