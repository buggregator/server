<template>
  <main class="profiler-event">
    <PageHeader button-title="Delete event" @delete="onDelete">
      <NuxtLink to="/">Home</NuxtLink>&nbsp;/
      <NuxtLink to="/profiler">Profiler</NuxtLink>&nbsp;/
      <NuxtLink :disabled="true">{{ eventId }}</NuxtLink>
    </PageHeader>

    <div v-if="pending && !event" class="profiler-event__loading">
      <div></div>
      <div></div>
      <div></div>
    </div>

    <ProfilerPage v-if="event" :event="event" />
  </main>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import { EventId } from "~/config/types";
import { useFetch, useNuxtApp, useRoute, useRouter } from "#app";
import { normalizeProfilerEvent } from "~/utils/normalize-event";
import ProfilerPage from "~/components/ProfilerPage/ProfilerPage.vue";
import PageHeader from "~/components/PageHeader/PageHeader.vue";

export default defineComponent({
  components: { ProfilerPage, PageHeader },
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
      title: `Profiler > ${this.eventId} | Buggregator`,
    };
  },
  computed: {
    event() {
      return this.serverEvent ? normalizeProfilerEvent(this.serverEvent) : null;
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
.profiler-event {
  @apply h-full w-full;

  > main {
    @apply flex flex-col md:flex-row;
  }

  .call-stack__wrapper {
    @apply w-full md:w-1/6 border-r border-gray-300 dark:border-gray-500;
  }

  .info__wrapper {
    @apply w-full h-full flex flex-col md:w-5/6 divide-y divide-gray-300 dark:divide-gray-500;
  }
}

.profiler-event__loading {
  @include loading;
}
</style>
