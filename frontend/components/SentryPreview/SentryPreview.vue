<template>
  <PreviewCard class="sentry-preview" :event="event">
    <SentryException :exception="exception" :max-frames="maxFrames">
      <NuxtLink tag="div" :to="eventLink" class="sentry-preview__link">
        <h3 class="sentry-preview__title">
          {{ exception.type }}
        </h3>

        <pre class="sentry-preview__text" v-html="exception.value" />
      </NuxtLink>
    </SentryException>
  </PreviewCard>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { NormalizedEvent } from "~/config/types";
import PreviewCard from "~/components/PreviewCard/PreviewCard.vue";
import SentryException from "~/components/SentryException/SentryException.vue";

export default defineComponent({
  components: {
    SentryException,
    PreviewCard,
  },
  props: {
    event: {
      type: Object as PropType<NormalizedEvent>,
      required: true,
    },
    maxFrames: {
      type: Number,
      default: 3,
    },
  },
  computed: {
    eventLink() {
      return `/sentry/${this.event.id}`;
    },
    exception() {
      const defaultException: object = {
        type: "Unknown",
        value: "Something went wrong",
        stacktrace: {
          frames: [],
        },
      };

      const eventExceptionValues = this.event?.payload?.exception?.values;

      return eventExceptionValues.length
        ? eventExceptionValues[0]
        : defaultException;
    },
  },
  methods: {
    isVisibleFrame(index: number): boolean {
      return index === 0;
    },
  },
});
</script>

<style lang="scss" scoped>
@import "assets/mixins";
.sentry-preview {
  @apply flex flex-col;
}

.sentry-preview__link {
  @apply cursor-pointer pb-2 flex-grow;
}

.sentry-preview__title {
  @apply mb-3 font-semibold;
}

.sentry-preview__text {
  @include text-muted;
  @apply text-sm break-all mb-3 p-3 dark:bg-gray-800;
}

.sentry-preview__frames {
  @apply border border-purple-200 dark:border-gray-600 flex-col justify-center w-full;
}
</style>
