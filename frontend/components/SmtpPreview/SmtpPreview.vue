<template>
  <PreviewCard class="smtp-preview" :event="event">
    <NuxtLink :to="eventLink" class="smtp-preview__link">
      <h3 class="smtp-preview__link-title">
        {{ event.payload.subject }}
      </h3>

      <div class="smtp-preview__link-text">
        <span><strong>To:</strong> {{ event.payload.to[0].email }} </span>

        <span>{{ dateFormat }}</span>
      </div>
    </NuxtLink>
  </PreviewCard>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import moment from "moment";
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
    dateFormat() {
      return moment(this.event.date).fromNow();
    },
    eventLink() {
      return `/smtp/${this.event.id}`;
    },
  },
});
</script>

<style lang="scss" scoped>
@import "assets/mixins";

.smtp-preview {
}

.smtp-preview__link {
  @apply block border dark:bg-gray-800 text-sm hover:bg-white dark:hover:bg-gray-900 flex items-stretch dark:border-gray-600 flex flex-col p-5;
}

.smtp-preview__nav-item {
  @apply border dark:border-0;
}

.smtp-preview__link-title {
  @apply font-semibold mb-2 dark:text-white;
}

.smtp-preview__link-text {
  @include text-muted;
  @apply flex justify-between text-xs;
}

.smtp-preview__link-text strong {
  @apply font-bold;
}
</style>
