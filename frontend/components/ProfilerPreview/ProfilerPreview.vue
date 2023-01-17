<template>
  <PreviewCard class="profiler-preview" :event="event">
    <NuxtLink tag="div" :to="eventLink" class="profiler-preview__link">
      <StatBoard :cost="event.payload.peaks" />
    </NuxtLink>
  </PreviewCard>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { NormalizedEvent } from "~/config/types";
import PreviewCard from "~/components/PreviewCard/PreviewCard.vue";
import StatBoard from "~/components/StatBoard/StatBoard.vue";

export default defineComponent({
  components: {
    PreviewCard,
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

.profiler-preview {
  @apply flex flex-col;
}

.profiler-preview__link {
  @apply cursor-pointer pb-2 flex-grow;
}
</style>
