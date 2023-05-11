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

      if (!$events?.items?.length) {
        $events.getAll();
      }

      return {
        events: $events.itemsGroupByType[EVENT_TYPES.HTTP_DUMP],
        title: "Http dumps",
        clearEvents: () => $events.removeByType(EVENT_TYPES.HTTP_DUMP),
      };
    }

    return {
      events: [],
      title: "Http dumps",
      clearEvents: () => {},
    };
  },
  head() {
    return {
      title: `Http dumps [${this.events.length}] | Buggregator`,
    };
  },
});
</script>
