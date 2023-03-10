<template>
  <main class="profiler-event">
    <PageProfiler v-if="event" :event="event" />
  </main>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import { EventId, Profiler, ServerEvent } from "~/config/types";
import { useNuxtApp, useRoute } from "#app";
import { normalizeProfilerEvent } from "~/utils/normalize-event";
import PageProfiler from "~/components/PageProfiler/PageProfiler.vue";

export default defineComponent({
  components: { PageProfiler },
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
  @apply relative h-full w-full;

  > main {
    @apply flex flex-col md:flex-row;
  }

  .call-stack__wrapper {
    @apply w-full md:w-1/6 border-r border-gray-300 dark:border-gray-500;
  }

  .info__wrapper {
    @apply w-full h-full flex flex-col md:w-5/6 divide-y divide-gray-300 dark:divide-gray-500;
  }
}
</style>
