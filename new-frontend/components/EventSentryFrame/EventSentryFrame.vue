<template>
  <div
    class="event-sentry-frame"
    :class="{ 'event-sentry-frame--empty': !hasBody }"
    @click="toggleOpen"
  >
    <div class="event-sentry-frame__head">
      <div class="event-sentry-frame__head-title">
        {{ frame.filename }}

        <span v-if="frame.function"> in {{ frame.function }} at line </span>

        {{ frame.lineno }}
      </div>

      <IconSvg
        v-if="frame.pre_context"
        class="event-sentry-frame__head-title-dd"
        :class="{ 'event-sentry-frame__head-title-dd--visible': open }"
        name="dd"
      />
    </div>

    <div v-if="open && hasBody" class="event-sentry-frame__body">
      <template v-if="frame.pre_context">
        <div
          v-for="(line, i) in frame.pre_context"
          :key="line"
          class="event-sentry-frame__body-line"
        >
          <div class="event-sentry-frame__body-line-position">
            {{ frame.lineno - (frame.pre_context.length - i) }}.
          </div>

          <pre class="event-sentry-frame__body-line-content" v-html="line" />
        </div>
      </template>

      <div
        v-if="frame.context_line"
        class="event-sentry-frame__body-line event-sentry-frame__body-line--selection"
      >
        <div class="event-sentry-frame__body-line-position">
          {{ frame.lineno }}.
        </div>

        <pre v-html="frame.context_line" />
      </div>

      <template v-if="frame.post_context">
        <div
          v-for="(line, i) in frame.post_context"
          :key="line"
          class="event-sentry-frame__body-line"
        >
          <div class="event-sentry-frame__body-line-position">
            {{ frame.lineno + i + 1 }}.
          </div>

          <pre class="event-sentry-frame__body-line-content" v-html="line" />
        </div>
      </template>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import IconSvg from "~/components/IconSvg/IconSvg.vue";

interface SentryFrame {
  filename: string;
  lineno: number;
  in_app: boolean;
  abs_path: string;
  pre_context: string[];
  context_line: string;
  post_context: string[];
}

export default defineComponent({
  components: {
    IconSvg,
  },
  props: {
    frame: {
      type: Object as PropType<SentryFrame>,
      required: true,
    },
    isOpen: {
      type: Boolean,
      default: true,
    },
  },
  data() {
    return {
      open: this.isOpen,
    };
  },
  computed: {
    hasBody() {
      return (
        this.frame.context_line ||
        this.frame.post_context ||
        this.frame.pre_context
      );
    },
  },
  methods: {
    toggleOpen(): void {
      if (this.hasBody) {
        this.open = !this.open;
      }
    },
  },
});
</script>

<style lang="scss" scoped>
@import "assets/mixins";

.event-sentry-frame {
  @apply text-xs border-b border-purple-200 dark:border-gray-600;
}

.event-sentry-frame__head {
  @apply bg-purple-50 dark:bg-gray-800 py-2 px-3 flex space-x-2 justify-between items-start cursor-pointer;

  .event-sentry-frame--empty & {
    @apply cursor-default;
  }
}

.event-sentry-frame__head-title {
  @include text-muted;
  @apply break-all font-semibold;
}

.event-sentry-frame__head-title-info {
  @apply text-gray-400;
}

.event-sentry-frame__head-title-dd {
  @apply w-5 h-4 flex justify-center border border-purple-300 shadow bg-white dark:bg-gray-600 py-1 rounded transform rotate-180;
}

.event-sentry-frame__head-title-dd--visible {
  @apply rotate-0;
}

.event-sentry-frame__body {
  @apply bg-gray-900 p-2 overflow-x-scroll;
}

.event-sentry-frame__body-line {
  @apply flex;
}
.event-sentry-frame__body-line--selection {
  @apply bg-pink-800 text-white;
}

.event-sentry-frame__body-line-position {
  @include text-muted;
  @apply w-12;

  .event-sentry-frame__body-line--selection & {
    @apply text-white;
  }
}

.event-sentry-frame__body-line-content {
  @apply text-gray-100;
}
</style>
