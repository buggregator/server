<template>
  <main class="sentry-event">
    <page-header button-title="Delete event" @delete="onDelete">
      <nuxt-link to="/">Home</nuxt-link>&nbsp;/
      <nuxt-link to="/sentry">Sentry</nuxt-link>&nbsp;/
      <nuxt-link :disabled="true">{{ eventId }}</nuxt-link>
    </page-header>

    <div v-if="pending" class="sentry-event__loading">
      <div></div>
      <div></div>
      <div></div>
    </div>

    <page-sentry v-if="event && pending !== true" :event="event" />
  </main>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import { EventId } from "~/config/types";
import { useNuxtApp, useRoute, useRouter, useFetch } from "#app";
import { normalizeSentryEvent } from "~/utils/normalize-event";
import PageSentry from "~/components/PageSentry/PageSentry.vue";
import PageHeader from "~/components/PageHeader/PageHeader.vue";

export default defineComponent({
  components: {
    PageSentry,
    PageHeader,
  },
  async setup() {
    const route = useRoute();
    const router = useRouter();
    const eventId = route.params.id as EventId;

    if (process.client) {
      const { $events } = useNuxtApp();
      const { data: event, pending } = await useFetch(
        $events.buildItemFetchUrl(eventId),
        {
          onResponse({ response }) {
            return response.data;
          },
          onResponseError() {
            router.push("/404");
          },
        }
      );

      return {
        serverEvent: event,
        pending,
        eventId,
        clearEvent: () => $events.removeById(eventId),
      };
    }

    return {
      serverEvent: null,
      pending: null,
      eventId,
      clearEvent: () => {},
    };
  },
  head() {
    const route = useRoute();

    return {
      title: `Sentry > ${route.params.id} | Buggregator`,
    };
  },
  computed: {
    event() {
      return this.serverEvent ? normalizeSentryEvent(this.serverEvent) : null;
    },
  },
  methods: {
    onDelete() {
      this.clearEvent();

      this.$router.push("/");
    },
  },
});
</script>

<style lang="scss" scoped>
@import "assets/mixins";

.sentry-event {
  @apply h-full w-full;
}

.sentry-event__loading {
  @include loading;
}
</style>
