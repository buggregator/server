<script lang="ts">
import { defineComponent, h } from "vue";
import PageIndex from "~/pages/index.vue";
import { EVENT_TYPES } from "~/config/constants";
import { useNuxtApp } from "#app";

export default defineComponent({
  extends: PageIndex,
  setup() {
    if (process.client) {
      const { $events } = useNuxtApp();
      return {
        events: $events.itemsGroupByType[EVENT_TYPES.SENTRY],
        title: "Sentry",
        clearEvents: () => $events.removeByType(EVENT_TYPES.SENTRY),
      };
    }

    return {
      events: [],
      title: "Sentry",
      clearEvents: () => {},
    };
  },
  head() {
    return {
      title: `Sentry [${this.events.length}] | Buggregator`,
    };
  },
});
</script>
