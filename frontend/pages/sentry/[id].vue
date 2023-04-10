<template>
  <main class="sentry-event">
    <PageHeader button-title="Delete event" @delete="onDelete">
      <NuxtLink to="/">Home</NuxtLink>&nbsp;/
      <NuxtLink to="/sentry">Sentry</NuxtLink>&nbsp;/
      <NuxtLink :disabled="true">{{ eventId }}</NuxtLink>
    </PageHeader>

    <div v-if="pending && !event" class="sentry-event__loading">
      <div></div>
      <div></div>
      <div></div>
    </div>

    <SentryPage v-if="event" :event="event" />
  </main>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import { EventId } from "~/config/types";
import { useNuxtApp, useRoute, useRouter, useFetch } from "#app";
import { normalizeSentryEvent } from "~/utils/normalize-event";
import SentryPage from "~/components/SentryPage/SentryPage.vue";
import PageHeader from "~/components/PageHeader/PageHeader.vue";

export default defineComponent({
  components: {
    SentryPage,
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
          onRequestError() {
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
      pending: false,
      eventId,
      clearEvent: () => {},
    };
  },
  head() {
    return {
      title: `Sentry > ${this.eventId} | Buggregator`,
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
