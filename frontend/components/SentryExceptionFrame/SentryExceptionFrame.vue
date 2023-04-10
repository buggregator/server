<template>
  <div
    class="sentry-exception-frame"
    :class="{ 'sentry-exception-frame--empty': !hasBody }"
  >
    <div class="sentry-exception-frame__head" @click="toggleOpen">
      <div class="sentry-exception-frame__head-title">
        {{ frame.filename }}

        <span v-if="frame.function"> in {{ frame.function }} at line </span>

        {{ frame.lineno }}
      </div>

      <IconSvg
        v-if="frame.pre_context"
        class="sentry-exception-frame__head-title-dd"
        :class="{
          'sentry-exception-frame__head-title-dd--visible': isFrameOpen,
        }"
        name="dd"
      />
    </div>

    <div v-if="isFrameOpen && hasBody" class="sentry-exception-frame__body">
      <template v-if="frame.pre_context">
        <div
          v-for="(line, i) in frame.pre_context"
          :key="line"
          class="sentry-exception-frame__body-line"
        >
          <div class="sentry-exception-frame__body-line-position">
            {{ frame.lineno - (frame.pre_context.length - i) }}.
          </div>

          <pre
            class="sentry-exception-frame__body-line-content"
            v-html="line"
          />
        </div>
      </template>

      <div
        v-if="frame.context_line"
        class="sentry-exception-frame__body-line sentry-exception-frame__body-line--selection"
      >
        <div class="sentry-exception-frame__body-line-position">
          {{ frame.lineno }}.
        </div>

        <pre v-html="frame.context_line" />
      </div>

      <template v-if="frame.post_context">
        <div
          v-for="(line, i) in frame.post_context"
          :key="line"
          class="sentry-exception-frame__body-line"
        >
          <div class="sentry-exception-frame__body-line-position">
            {{ frame.lineno + i + 1 }}.
          </div>

          <pre
            class="sentry-exception-frame__body-line-content"
            v-html="line"
          />
        </div>
      </template>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import IconSvg from "~/components/IconSvg/IconSvg.vue";

export interface SentryFrame {
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
      isFrameOpen: this.isOpen,
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
        this.isFrameOpen = !this.isFrameOpen;
      }
    },
  },
});
</script>

<style lang="scss" scoped>
@import "assets/mixins";

.sentry-exception-frame {
  @apply text-xs border-b border-purple-200 dark:border-gray-600;
}

.sentry-exception-frame__head {
  @apply bg-purple-50 dark:bg-gray-800 py-2 px-3 flex space-x-2 justify-between items-start cursor-pointer;

  .sentry-exception-frame--empty & {
    @apply cursor-default;
  }
}

.sentry-exception-frame__head-title {
  @include text-muted;
  @apply break-all font-semibold;
}

.sentry-exception-frame__head-title-info {
  @apply text-gray-400;
}

.sentry-exception-frame__head-title-dd {
  @apply w-5 h-4 flex justify-center border border-purple-300 shadow bg-white dark:bg-gray-600 py-1 rounded transform rotate-180;
}

.sentry-exception-frame__head-title-dd--visible {
  @apply rotate-0;
}

.sentry-exception-frame__body {
  @apply bg-gray-900 p-2 overflow-x-scroll;
}

.sentry-exception-frame__body-line {
  @apply flex;
}

.sentry-exception-frame__body-line--selection {
  @apply bg-pink-800 text-white;
}

.sentry-exception-frame__body-line-position {
  @include text-muted;
  @apply w-12 select-none;

  .sentry-exception-frame__body-line--selection & {
    @apply text-white;
  }
}

.sentry-exception-frame__body-line-content {
  @apply text-gray-100;
}
</style>
