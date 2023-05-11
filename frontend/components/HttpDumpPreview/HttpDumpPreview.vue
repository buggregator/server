<template>
  <PreviewCard class="http-dumps-preview" :event="event">
    <NuxtLink tag="div" :to="eventLink" class="http-dumps-preview__link">
      <strong>{{ event.payload.request.method }}</strong>: <code>{{ uri }}</code>
    </NuxtLink>
  </PreviewCard>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { NormalizedEvent } from "~/config/types";
import PreviewCard from "~/components/PreviewCard/PreviewCard.vue";

export default defineComponent({
  components: {
    PreviewCard,
  },
  props: {
    event: {
      type: Object as PropType<NormalizedEvent>,
      required: true,
    },
  },
  computed: {
    eventLink() {
      return `/http-dumps/${this.event.id}`;
    },
    uri() {
      return decodeURI(this.event.payload.request.uri);
    },
  },
});
</script>

<style lang="scss" scoped>
@import "assets/mixins";

.http-dumps-preview {
  @apply flex flex-col;
}

.http-dumps-preview__link {
  @apply cursor-pointer flex-grow p-3 bg-gray-200 dark:bg-gray-800;
}
</style>
