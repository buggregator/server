<script lang="ts">
import { defineComponent } from "vue";
import PageIndex from "~/pages/index.vue";
import { EVENT_TYPES } from "~/config/constants";
import { useNuxtApp } from "#app";

export default defineComponent({
  extends: PageIndex,
  setup() {
    if (process.client) {
      const { $events } = useNuxtApp();

      return {
        events: $events.itemsGroupByType[EVENT_TYPES.PROFILER],
        clearEvents: () => $events.removeByType(EVENT_TYPES.PROFILER),
      };
    }

    return {
      events: [],
      clearEvents: () => {},
    };
  },
  head() {
    return {
      title: `Profiler [${this.events.length}] | Buggregator`,
    };
  },
});
</script>
