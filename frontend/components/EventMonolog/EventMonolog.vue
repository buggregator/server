<template>
  <event-card class="event-monolog" :event="event">
    <CodeSnippet class="event-monolog__snippet" :code="event.payload.message" />

    <CodeSnippet
      v-if="event.payload.context"
      class="event-monolog__snippet"
      language="json"
      :code="event.payload.context"
    />

    <CodeSnippet
      v-for="(field, key) in event.payload.extra"
      :key="key"
      class="event-monolog__snippet"
      :code="{ [key]: field }"
    />
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
.event-monolog {
  @apply text-xs break-all;
}

.event-monolog__snippet {
  @apply relative bg-gray-200 dark:bg-gray-800 border-b-0 mt-0 text-white break-words;

  & + & {
    @apply border-t-2;
  }
}
</style>
