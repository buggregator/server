<template>
  <PreviewCard class="preview-fallback" :event="event">
    <CodeSnippet class="preview-fallback__snippet" :code="event" />
  </PreviewCard>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { NormalizedEvent } from "~/config/types";
import PreviewCard from "~/components/PreviewCard/PreviewCard.vue";
import CodeSnippet from "~/components/codeSnippet/CodeSnippet.vue";

export default defineComponent({
  components: {
    PreviewCard,
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
.preview-fallback {
  @apply text-xs break-all;
}

.preview-fallback__snippet {
  @apply relative bg-gray-200 dark:bg-gray-800 border-b-0 mt-0 text-white break-words;
}
</style>
