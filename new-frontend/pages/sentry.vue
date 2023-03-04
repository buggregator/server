<script lang="ts">
import { defineComponent } from "vue";
import { storeToRefs } from "pinia";
import { useThemeStore } from "~/stores/theme";
import PageIndex from "~/pages/index.vue";
import { EVENT_TYPES } from "~/config/constants";
import { useNuxtApp } from "#app";

export default defineComponent({
  extends: PageIndex,
  setup() {
    const themeStore = useThemeStore();
    const { themeType } = storeToRefs(themeStore);

    const { $events } = useNuxtApp();

    return {
      events: $events.getItemsByType(EVENT_TYPES.SENTRY),
      themeType,
      removeEventsByType: $events.removeByType,
    };
  },
  methods: {
    clearEvents() {
      this.removeEventsByType(EVENT_TYPES.SENTRY);
    },
  },
});
</script>

<style lang="scss" scoped>
@import "assets/mixins";
.events-page {
  @apply h-full w-full;
}

.events-page__header {
  @include border-style;
  @apply md:sticky md:top-0 z-50 bg-white dark:bg-gray-900 border-b flex justify-between items-center px-2;
}

.events-page__filters {
  @include border-style;
  @apply flex flex-col py-2 md:flex-row justify-center md:justify-between items-center gap-2;
}

.events-page__events {
  @include border-style;
  @apply flex flex-col divide-y;
}

.events-page__event {
  & + & {
    @apply border-b;
  }
}

.events-page__welcome {
  @apply flex-1 p-4 flex flex-col justify-center items-center bg-gray-50 dark:bg-gray-800 w-full h-full;
}

.events-page__btn-clear {
  @apply px-3 py-1 text-xs bg-red-800 text-white rounded-sm hover:bg-red-700 transition transition-all duration-300;
}
</style>
