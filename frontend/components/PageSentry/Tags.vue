<template>
  <section class="sentry-tags">
    <h3 class="sentry-section__title">tags</h3>

    <div class="sentry-tags__boxes">
      <div
          v-if="event.contexts.runtime?.name"
          class="sentry-tags__box">
        <span class="sentry-tags__box-title">runtime</span>
        <h4 class="sentry-tags__box-name">{{ event.contexts.runtime.name }}</h4>
        <p class="sentry-tags__box-value">Version: {{ event.contexts.runtime.version }}</p>
      </div>
      <div
          v-if="event.contexts.os?.name"
          class="sentry-tags__box">
        <span class="sentry-tags__box-title">os</span>
        <h4 class="sentry-tags__box-name">{{ event.contexts.os.name }}</h4>
        <p class="sentry-tags__box-value">Version: {{ event.contexts.os.version }}</p>
      </div>

      <div
          v-if="event.sdk.name"
          class="sentry-tags__box">
        <span class="sentry-tags__box-title">sdk</span>
        <h4 class="sentry-tags__box-name">{{ event.sdk.name }}</h4>
        <p class="sentry-tags__box-value">Version: {{ event.sdk.version }}</p>
      </div>
    </div>

    <div class="sentry-tags__labels">
      <div class="sentry-tags__label">
        <div class="sentry-tags__label-name">env</div>
        <div class="sentry-tags__label-value">
          {{ event.environment }}
        </div>
      </div>
      <div
          v-if="event.logger"
          class="sentry-tags__label">
        <div class="sentry-tags__label-name">logger</div>
        <div class="sentry-tags__label-value">
          {{ event.logger }}
        </div>
      </div>
      <div
          v-if="event.contexts.os?.name"
          class="sentry-tags__label">
        <div class="sentry-tags__label-name">os</div>
        <div class="sentry-tags__label-value">
          {{ event.contexts.os.name }} {{ event.contexts.os.version }}
        </div>
      </div>
      <div
          v-if="event.contexts.runtime?.name"
          class="sentry-tags__label">
        <div class="sentry-tags__label-name">runtime</div>
        <div class="sentry-tags__label-value">
          {{ event.contexts.runtime.name }} {{ event.contexts.runtime.version }}
        </div>
      </div>
      <div
          v-if="event.serverName"
          class="sentry-tags__label">
        <div class="sentry-tags__label-name">server name</div>
        <div class="sentry-tags__label-value">
          {{ event.serverName }}
        </div>
      </div>
    </div>

    <CodeSnippet
        v-if="event.tags"
        class="mt-3"
        language="json"
        :code="event.tags"/>
  </section>
</template>

<script lang="ts">
import {defineComponent, PropType} from "vue";
import {Sentry} from "~/config/types";
import CodeSnippet from "~/components/CodeSnippet/CodeSnippet.vue";

export default defineComponent({
  components: {
    CodeSnippet
  },
  props: {
    event: {
      type: Object as PropType<Sentry>,
      required: true,
    },
  },
});
</script>

<style lang="scss" scoped>
@import "assets/mixins";

.sentry-tags {
}

.sentry-section__title {
  @include text-muted;
  @apply font-bold uppercase text-sm mb-5;
}

.sentry-tags__boxes {
  @apply flex items-stretch flex-col md:flex-row mb-5 gap-x-4;
}

.sentry-tags__box {
  @apply border border-purple-300 dark:border-gray-400 rounded px-4 pb-2 pt-1 hover:bg-purple-50 dark:hover:bg-purple-600 cursor-pointer mb-3 md:mb-0;
}

.sentry-tags__box-title {
  @include text-muted;
  @apply text-xs font-bold;
}

.sentry-tags__box-name {
  @apply font-bold;
}

.sentry-tags__box-value {
  @apply text-sm;
}

.sentry-tags__labels {
  @apply flex flex-row flex-wrap items-center text-purple-600 dark:text-purple-100 gap-3;
}

.sentry-tags__label {
  @apply flex border border-purple-300 rounded text-xs items-center;
}

.sentry-tags__label-name {
  @apply px-3 py-1 border-r;
}

.sentry-tags__label-value {
  @apply px-3 py-1 bg-purple-100 dark:bg-purple-800 rounded-r font-bold;
}
</style>