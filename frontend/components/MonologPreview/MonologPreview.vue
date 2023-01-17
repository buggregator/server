<template>
  <PreviewCard class="monolog-preview" :event="event">
    <CodeSnippet
      class="monolog-preview__snippet"
      :code="event.payload.message"
    />

    <CodeSnippet
      v-if="event.payload.context"
      class="monolog-preview__snippet"
      language="json"
      :code="event.payload.context"
    />

    <CodeSnippet
      v-for="(field, key) in event.payload.extra"
      :key="key"
      class="monolog-preview__snippet"
      :code="{ [key]: field }"
    />
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
.monolog-preview {
  @apply text-xs break-all;
}

.monolog-preview__snippet {
  @apply relative bg-gray-200 dark:bg-gray-800 border-b-0 mt-0 text-white break-words;

  & + & {
    @apply border-t-2;
  }
}
</style>
