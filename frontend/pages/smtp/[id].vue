<template>
  <main class="smtp-event">
    <PageHeader button-title="Delete event" @delete="onDelete">
      <NuxtLink to="/">Home</NuxtLink>&nbsp;/
      <NuxtLink to="/smtp">Smtp</NuxtLink>&nbsp;/
      <NuxtLink :disabled="true">{{ eventId }}</NuxtLink>
    </PageHeader>

    <div v-if="pending && !event" class="smtp-event__loading">
      <div></div>
      <div></div>
      <div></div>
    </div>

    <SmtpPage v-if="event" :event="event" :html-source="html" />
  </main>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import { EventId } from "~/config/types";
import { useFetch, useNuxtApp, useRoute, useRouter } from "#app";
import { normalizeSMTPEvent } from "~/utils/normalize-event";
import SmtpPage from "~/components/SmtpPage/SmtpPage.vue";
import PageHeader from "~/components/PageHeader/PageHeader.vue";
import { REST_API_URL } from "~/utils/events-transport";

export default defineComponent({
  components: { SmtpPage, PageHeader },
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
        html: `<iframe src="${REST_API_URL}/api/smtp/${eventId}/html"/>`,
        clearEvent: () => $events.removeById(eventId),
      };
    }

    return {
      serverEvent: null,
      pending: false,
      eventId,
      html: "",
      clearEvent: () => {},
    };
  },
  head() {
    return {
      title: `SMTP > ${this.eventId} | Buggregator`,
    };
  },
  computed: {
    event() {
      return this.serverEvent ? normalizeSMTPEvent(this.serverEvent) : null;
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

.smtp-event {
  @apply flex-1 flex flex-col h-full w-full;
}

.smtp-event__loading {
  @include loading;
}
</style>
