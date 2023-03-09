<template>
  <event-card class="event-inspector" :event="event">
    <NuxtLink tag="div" :to="eventLink" class="event-inspector__link">
      <stat-board :transaction="event.payload[0]" />
    </NuxtLink>
  </event-card>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { NormalizedEvent } from "~/config/types";
import EventCard from "~/components/EventCard/EventCard.vue";
import StatBoard from "~/components/PageInspector/StatBoard.vue";

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
      return `/inspector/${this.event.id}`;
    },
  },
});
</script>

<style lang="scss" scoped>
@import "assets/mixins";

.event-inspector {
  @apply flex flex-col;
}

.event-inspector__link {
  @apply cursor-pointer pb-2 flex-grow;
}
</style>
