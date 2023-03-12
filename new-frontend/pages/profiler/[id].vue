<template>
  <main class="profiler-event">
    <page-header button-title="Delete event" @delete="clearEvent">
      Profiler event: {{ event.id }}
    </page-header>

    <page-profiler v-if="event" :event="event" />
  </main>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import { EventId, Profiler, ServerEvent } from "~/config/types";
import { useNuxtApp, useRoute } from "#app";
import { normalizeProfilerEvent } from "~/utils/normalize-event";
import PageProfiler from "~/components/PageProfiler/PageProfiler.vue";
import PageHeader from "~/components/PageHeader/PageHeader.vue";

export default defineComponent({
  components: { PageProfiler, PageHeader },
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
