<template>
  <main class="inspector-event">
    <page-header button-title="Delete event" @delete="onDelete">
      <nuxt-link to="/">Home</nuxt-link>&nbsp;/
      <nuxt-link to="/inspector">Inspector</nuxt-link>&nbsp;/
      <nuxt-link :disabled="true">{{ event.id }}</nuxt-link>
    </page-header>

    <div v-if="pending && !event" class="inspector-event__loading">
      <div></div>
      <div></div>
      <div></div>
    </div>

    <page-inspector v-if="event" :event="event" />
  </main>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import { EventId } from "~/config/types";
import { useFetch, useNuxtApp, useRoute, useRouter } from "#app";
import { normalizeInspectorEvent } from "~/utils/normalize-event";
import PageInspector from "~/components/PageInspector/PageInspector.vue";
import PageHeader from "~/components/PageHeader/PageHeader.vue";

export default defineComponent({
  components: { PageInspector, PageHeader },

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
      title: `Inspector > ${this.eventId} | Buggregator`,
    };
  },
  computed: {
    event() {
      return this.serverEvent
        ? normalizeInspectorEvent(this.serverEvent)
        : null;
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
.inspector-event {
  @apply h-full w-full;
}
.inspector-event__loading {
  @include loading;
}
</style>
