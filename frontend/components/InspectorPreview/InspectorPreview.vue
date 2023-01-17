<template>
  <PreviewCard class="inspector-preview" :event="event">
    <NuxtLink tag="div" :to="eventLink" class="inspector-preview__link">
      <InspectorStatBoard :transaction="event.payload[0]" />
    </NuxtLink>
  </PreviewCard>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { NormalizedEvent } from "~/config/types";
import PreviewCard from "~/components/PreviewCard/PreviewCard.vue";
import InspectorStatBoard from "~/components/InspectorStatBoard/InspectorStatBoard.vue";

export default defineComponent({
  components: {
    PreviewCard,
    InspectorStatBoard,
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

.inspector-preview {
  @apply flex flex-col;
}

.inspector-preview__link {
  @apply cursor-pointer pb-2 flex-grow;
}
</style>
