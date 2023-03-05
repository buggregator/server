<template>
  <main class="profiler-event">
    <EventProfiler v-if="event" :event="event" />
  </main>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import { EventId, Profiler, ServerEvent } from "~/config/types";
import { useNuxtApp, useRoute } from "#app";
import { normalizeProfilerEvent } from "~/utils/normalize-event";
import EventProfiler from "~/components/EventProfiler/EventProfiler.vue";

export default defineComponent({
  components: { EventProfiler },
  setup() {
    const route = useRoute();
    const eventId = route.params.id as EventId;

    if (process.client) {
      const { $events } = useNuxtApp();
      const serverEvent = $events.getItemById(
        eventId
      ) as ServerEvent<Profiler> | null;

      return {
        event: serverEvent ? normalizeProfilerEvent(serverEvent) : null,
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
      title: `Profiler > ${route.params.id} | Buggregator`,
    };
  },
});
</script>

<style lang="scss" scoped>
.profiler-event {
  @apply h-full w-full;
}
</style>
