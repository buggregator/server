<template>
  <main class="smtp-event">
    <page-header button-title="Delete event" @delete="clearEvent">
      <nuxt-link to="/">Home</nuxt-link>&nbsp;/
      <nuxt-link to="/smtp">Smtp</nuxt-link>&nbsp;/
      <nuxt-link :disabled="true">{{ event.id }}</nuxt-link>
    </page-header>

    <page-smtp v-if="event" :event="event" :html-source="html" />
  </main>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import { EventId, SMTP, ServerEvent } from "~/config/types";
import { useNuxtApp, useRoute } from "#app";
import { normalizeSMTPEvent } from "~/utils/normalize-event";
import PageSmtp from "~/components/PageSmtp/PageSmtp.vue";
import PageHeader from "~/components/PageHeader/PageHeader.vue";
import { REST_API_URL } from "~/utils/events-transport";

export default defineComponent({
  components: { PageSmtp, PageHeader },
  setup() {
    const route = useRoute();
    const eventId = route.params.id as EventId;

    if (process.client) {
      const { $events } = useNuxtApp();
      const serverEvent = $events.getItemById(
        eventId
      ) as ServerEvent<SMTP> | null;

      return {
        event: serverEvent ? normalizeSMTPEvent(serverEvent) : null,
        html: `<iframe src="${REST_API_URL}/api/smtp/${eventId}/html"/>`,
        clearEvent: () => $events.removeById(eventId),
      };
    }

    return {
      event: null,
      html: "",
      clearEvent: () => {},
    };
  },
  head() {
    const route = useRoute();

    return {
      title: `SMTP > ${route.params.id} | Buggregator`,
    };
  },
});
</script>

<style lang="scss" scoped>
.smtp-event {
  @apply h-full w-full;
}
</style>
