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

      if (!$events?.items?.value.length) {
        $events.getAll();
      }

      return {
        events: $events.getItemsByType(EVENT_TYPES.SMTP),
        clearEvents: () => $events.removeByType(EVENT_TYPES.SMTP),
      };
    }
    return {};
  },
});
</script>
