<template>
  <event-card class="event-profiler" :event="event">
    <NuxtLink tag="div" :to="eventLink" class="event-profiler__link">
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

.event-profiler {
  @apply flex flex-col;
}

.event-profiler__link {
  @apply cursor-pointer pb-2 flex-grow;
}
</style>
