<template>
  <event-card class="event-fallback" :event="event">
    <CodeSnippet class="event-fallback__snippet" :code="event" />
  </event-card>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { NormalizedEvent } from "~/config/types";
import EventCard from "~/components/EventCard/EventCard.vue";
import CodeSnippet from "~/components/CodeSnippet/CodeSnippet.vue";

export default defineComponent({
  components: {
    EventCard,
    CodeSnippet,
  },
  props: {
    event: {
      type: Object as PropType<NormalizedEvent>,
      required: true,
    },
  },
});
</script>

<style lang="scss" scoped>
.event-fallback {
  @apply text-xs break-all;
}

.event-fallback__snippet {
  @apply relative bg-gray-200 dark:bg-gray-800 border-b-0 mt-0 text-white break-words;
}
</style>
