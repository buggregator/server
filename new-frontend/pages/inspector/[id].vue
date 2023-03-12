<template>
  <main class="inspector-event">
    <PageInspector v-if="event" :event="event" @delete="clearEvent" />
  </main>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import { EventId, Inspector, ServerEvent } from "~/config/types";
import { useNuxtApp, useRoute } from "#app";
import { normalizeInspectorEvent } from "~/utils/normalize-event";
import PageInspector from "~/components/PageInspector/PageInspector.vue";

export default defineComponent({
  components: { PageInspector },
  setup() {
    const route = useRoute();
    const eventId = route.params.id as EventId;

    if (process.client) {
      const { $events } = useNuxtApp();
      const serverEvent = $events.getItemById(
        eventId
      ) as ServerEvent<Inspector> | null;

      return {
        event: serverEvent ? normalizeInspectorEvent(serverEvent) : null,
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
      title: `Inspector > ${route.params.id} | Buggregator`,
    };
  },
});
</script>

<style lang="scss" scoped>
.inspector-event {
  @apply h-full w-full;
}
</style>
