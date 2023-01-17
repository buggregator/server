<template>
  <div class="sentry-page">
    <main class="sentry-page__main">
      <header class="sentry-page__main-header">
        <h1 class="sentry-page__main-exception">{{ exception }}</h1>

        <pre class="sentry-page__main-exception-message" v-html="message" />
        <p class="sentry-page__main-date">{{ date }}</p>
      </header>

      <SentryPageTags :event="event.payload" class="sentry-page__section" />

      <section class="sentry-page__section">
        <h3 class="sentry-page__section-title">exceptions</h3>

        <div class="sentry-page__section-exceptions">
          <SentryException
            v-for="e in event.payload.exception.values"
            :key="`exception-${e.value} - ${e.type}`"
            :exception="e"
          />
        </div>
      </section>

      <SentryPageBreadcrumbs
        :event="event.payload"
        class="sentry-page__section"
      />
      <SentryPageRequest :event="event.payload" class="sentry-page__section" />
      <SentryPageApp :event="event.payload" class="sentry-page__section" />
      <SentryPageDevice :event="event.payload" class="sentry-page__section" />
    </main>
  </div>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { NormalizedEvent } from "~/config/types";
import SentryException from "~/components/SentryException/SentryException.vue";
import SentryPageBreadcrumbs from "~/components/SentryPageBreadcrumbs/SentryPageBreadcrumbs.vue";
import SentryPageTags from "~/components/SentryPageTags/SentryPageTags.vue";
import SentryPageRequest from "~/components/SentryPageRequest/SentryPageRequest.vue";
import SentryPageApp from "~/components/SentryPageApp/SentryPageApp.vue";
import SentryPageDevice from "~/components/SentryPageDevice/SentryPageDevice.vue";
import moment from "moment";

export default defineComponent({
  components: {
    SentryPageTags,
    SentryPageBreadcrumbs,
    SentryPageRequest,
    SentryPageApp,
    SentryPageDevice,
    SentryException,
  },
  props: {
    event: {
      type: Object as PropType<NormalizedEvent>,
      required: true,
    },
  },
  computed: {
    mainException() {
      return this.event.payload.exception.values[0];
    },
    exception(): string {
      return this.mainException.type;
    },
    message(): string {
      return this.mainException.value;
    },
    date(): string {
      return moment(this.event.payload.timestamp).toLocaleString();
    },
  },
});
</script>

<style lang="scss" scoped>
@import "assets/mixins";

.sentry-page {
  @apply relative;
}

.sentry-page__main {
  @apply flex flex-col w-full;
}

.sentry-page__main-header {
  @apply bg-gray-50 dark:bg-gray-900 py-5 px-4 md:px-6 lg:px-8;
}

.sentry-page__main-exception {
  @apply font-bold text-sm sm:text-base md:text-lg lg:text-2xl break-all sm:break-normal mb-3;
}

.sentry-page__main-date {
  @include text-muted;
  @apply text-xs mt-3;
}

.sentry-page__main-exception-message {
  @apply text-sm;
}

.sentry-page__section {
  @apply py-5 px-4;
}

.sentry-page__section-exceptions {
  @apply flex flex-col gap-y-10;
}

.sentry-page__section-title {
  @include text-muted;
  @apply font-bold uppercase text-sm mb-5;
}
</style>
