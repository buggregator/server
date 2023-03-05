<template>
  <event-card class="event-sentry" :event="event">
    <NuxtLink tag="div" :to="eventLink" class="event-sentry__link">
      <h3 class="event-sentry__title">
        {{ exception.type }}
      </h3>

      <pre class="event-sentry__text" v-html="exception.value" />
    </NuxtLink>

    <div v-if="exceptionFrames.length" class="event-sentry__files">
      <template v-for="(frame, i) in exceptionFrames" :key="frame.context_line">
        <event-sentry-frame :frame="frame" :is-open="isVisibleFrame(i)" />
      </template>
    </div>
  </event-card>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { NormalizedEvent } from "~/config/types";
import EventCard from "~/components/EventCard/EventCard.vue";
import EventSentryFrame from "~/components/EventSentryFrame/EventSentryFrame.vue";

export default defineComponent({
  components: {
    EventSentryFrame,
    EventCard,
  },
  props: {
    event: {
      type: Object as PropType<NormalizedEvent>,
      required: true,
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
    exceptionFrames(): object[] {
      return this.exception.stacktrace.frames || [];
    },
  },
  methods: {
    isVisibleFrame(index): boolean {
      return index === 0;
    },
  },
});
</script>

<style lang="scss" scoped>
@import "assets/mixins";

.event-sentry {
  @apply flex flex-col;
}

.event-sentry__link {
  @apply cursor-pointer pb-2 flex-grow;
}

.event-sentry__title {
  @apply mb-3 font-semibold;
}

.event-sentry__text {
  @include text-muted;
  @apply text-sm break-all mb-3 p-3 dark:bg-gray-800;
}

.event-sentry__files {
  @apply border border-purple-200 dark:border-gray-600 flex-col justify-center w-full;
}
</style>
