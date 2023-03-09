<template>
  <div class="sentry-exception">
    <slot>
      <h3 class="sentry-exception__title">
        {{ exception.type }}
      </h3>

      <pre class="sentry-exception__text" v-html="exception.value"/>
    </slot>

    <div v-if="exceptionFrames.length" class="sentry-exception__frames">
      <template v-for="(frame, i) in exceptionFrames" :key="frame.context_line">
        <sentry-frame :frame="frame" :is-open="isVisibleFrame(i)"/>
      </template>
    </div>
  </div>
</template>

<script lang="ts">
import {defineComponent, PropType} from "vue";
import {SentryException, SentryFrame as SentryFrameType} from "~/config/types";
import SentryFrame from "./SentryFrame.vue";

export default defineComponent({
  components: {
    SentryFrame,
  },
  props: {
    exception: {
      type: Object as PropType<SentryException>,
      required: true,
    },
    maxFrames: {
      type: Number,
      default: 0,
    }
  },
  computed: {
    exceptionFrames(): SentryFrameType[] {
      const frames = (this.exception.stacktrace.frames || []);

      if (this.maxFrames > 0) {
        return frames.reverse().slice(0, this.maxFrames);
      }

      return frames;
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

.sentry-exception {
  @apply flex flex-col;
}

.sentry-exception__link {
  @apply cursor-pointer pb-2 flex-grow;
}

.sentry-exception__text {
  @include text-muted;
  @apply text-sm break-all mb-3 p-3 dark:bg-gray-800;
}

.sentry-exception__title {
  @apply mb-3 font-semibold;
}

.sentry-exception__text {
  @include text-muted;
  @apply text-sm break-all mb-3 p-3 dark:bg-gray-800;
}

.sentry-exception__frames {
  @apply border border-purple-200 dark:border-gray-600 flex-col justify-center w-full;
}
</style>
