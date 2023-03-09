<template>
  <div class="page-sentry">
    <main class="page-sentry__main">
      <header class="page-sentry__main-header">
        <h1 class="page-sentry__main-exception">{{ exception }}</h1>

        <pre class="page-sentry__main-exception-message" v-html="message"/>
        <p class="page-sentry__main-date">{{ date }}</p>
      </header>

      <Tags :event="event.payload" class="sentry-section"/>

      <section class="sentry-section">
        <h3 class="sentry-section__title">exceptions</h3>

        <div class="sentry-section__exceptions">
          <SentryException  v-for="e in event.payload.exception.values" :key="`exception-${e.value} - ${e.type}`" :exception="e"/>
        </div>
      </section>


      <Breadcrumbs :event="event.payload" class="sentry-section"/>
<!--      <User :event="event.payload" class="sentry-section"/>-->
      <Request :event="event.payload" class="sentry-section"/>
      <App :event="event.payload" class="sentry-section"/>
      <Device :event="event.payload" class="sentry-section"/>
<!--      <OS :event="event.payload" class="sentry-section"/>-->
    </main>
  </div>
</template>

<script lang="ts">
import {defineComponent, PropType} from "vue";
import {NormalizedEvent, Sentry} from "~/config/types";
import SentryException from "~/components/SentryException/SentryException.vue";
import Breadcrumbs from "~/components/PageSentry/Breadcrumbs.vue";
import Tags from "~/components/PageSentry/Tags.vue";
import Request from "~/components/PageSentry/Request.vue";
import App from "~/components/PageSentry/App.vue";
import Device from "~/components/PageSentry/Device.vue";
import moment from "moment";

export default defineComponent({
  components: {
    Tags, Breadcrumbs, Request, App, Device, SentryException
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
      return moment(this.event.payload.timestamp).toLocaleString()
    }
  }
});
</script>

<style lang="scss" scoped>
@import "assets/mixins";

.page-sentry {
  @apply relative;
}

.page-sentry__main {
  @apply flex flex-col w-full;
}

.page-sentry__main-header {
  @apply bg-gray-50 dark:bg-gray-900 py-5 px-4 md:px-6 lg:px-8;
}

.page-sentry__main-exception {
  @apply font-bold text-sm sm:text-base md:text-lg lg:text-2xl break-all sm:break-normal mb-3;
}

.page-sentry__main-date {
  @include text-muted;
  @apply text-xs mt-3;
}

.page-sentry__main-exception-message {
  @apply text-sm;
}

.sentry-section {
  @apply py-5 px-4;
}

.sentry-section__exceptions {
  @apply flex flex-col gap-y-10;
}

.sentry-section__title {
  @include text-muted;
  @apply font-bold uppercase text-sm mb-5;
}
</style>
