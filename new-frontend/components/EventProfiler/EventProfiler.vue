<template>
  <event-card class="event-sentry" :event="event">
    <NuxtLink tag="div" :to="eventLink" class="event-sentry__link">
      <stat-board :cost="event.payload.peaks" />
    </NuxtLink>
  </event-card>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { NormalizedEvent } from "~/config/types";
import EventCard from "~/components/EventCard/EventCard.vue";
import StatBoard from "~/components/StatBoard/StatBoard.vue";

export default defineComponent({
  components: {
    EventCard,
    StatBoard,
  },
  props: {
    event: {
      type: Object as PropType<NormalizedEvent>,
      required: true,
    },
  },
  computed: {
    eventLink() {
      return `/profiler/${this.event.id}`;
    },
  },
});
</script>

<style lang="scss" scoped>
@import "assets/mixins";

.event-sentry {
  @apply flex flex-col;
}

.event-sentry__link {
  @apply cursor-pointer pb-2 flex-grow;
}

.event-sentry__title {
  @apply mb-3 font-semibold;
}

.event-sentry__text {
  @include text-muted;
  @apply text-sm break-all mb-3 p-3 dark:bg-gray-800;
}

.event-sentry__files {
  @apply border border-purple-200 dark:border-gray-600 flex-col justify-center w-full;
}
</style>
