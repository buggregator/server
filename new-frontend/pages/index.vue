<template>
  <div class="events-page">
    <header class="events-page__header">
      <div v-if="events.length" class="events-page__filters">
        <button class="events-page__btn-clear" @click="clearEvents">
          Clear screen
        </button>
      </div>
    </header>

    <main v-if="events.length" class="events-page__events">
      <event-mapper
        v-for="event in events"
        :key="event.uuid"
        :event="event"
        class="events-page__event"
      />
    </main>

    <section v-if="!events.length" class="events-page__welcome">
      <layout-tips class="events-page__tips" />
    </section>
  </div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import EventMapper from "~/components/EventMapper/EventMapper.vue";
import { storeToRefs } from "pinia";
import { useThemeStore, THEME_MODES } from "~/stores/theme";
import { useEventStore } from "~/stores/events";
import LayoutTips from "~/components/LayoutTips/LayoutTips.vue";

export default defineComponent({
  components: {
    LayoutTips,
    EventMapper,
  },
  setup() {
    const themeStore = useThemeStore();
    const { themeType } = storeToRefs(themeStore);

    const eventsStore = useEventStore();
    const { removeAllEvents, removeEventByUuid } = eventsStore;
    const { events } = storeToRefs(eventsStore);

    return {
      events,
      themeType,
      removeAllEvents,
      removeEventByUuid,
    };
  },
  mounted() {
    if (this.themeType === THEME_MODES.DARK) {
      document?.documentElement?.classList?.add(THEME_MODES.DARK);
    } else {
      document?.documentElement?.classList?.remove(THEME_MODES.DARK);
    }
  },
  methods: {
    clearEvents() {
      this.removeAllEvents();
    },
  },
});
</script>

<style lang="scss" scoped>
@import "assets/mixins";
.events-page {
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
  @apply flex-1 p-4 flex flex-col justify-center items-center bg-gray-50 dark:bg-gray-800;
}

.events-page__btn-clear {
  @apply px-3 py-1 text-xs bg-red-800 text-white rounded-sm hover:bg-red-700 transition transition-all duration-300;
}
</style>
