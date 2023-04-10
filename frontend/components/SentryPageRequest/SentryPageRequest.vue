<template>
  <section v-if="hasRequest" class="sentry-page-request">
    <h3 class="sentry-page-request__title">request</h3>

    <h3 class="sentry-page-request__url">
      <strong>{{ event.request.method }}:</strong> {{ event.request.url }}
    </h3>

    <h3 class="sentry-page-request__title sentry-page-request__title--sub">
      headers
    </h3>
    <EventTable>
      <EventTableRow
        v-for="(value, title) in headers"
        :key="title"
        :title="title"
      >
        {{ value[0] || value }}
      </EventTableRow>
    </EventTable>
  </section>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { Sentry } from "~/config/types";
import EventTable from "~/components/EventTable/EventTable.vue";
import EventTableRow from "~/components/EventTableRow/EventTableRow.vue";

export default defineComponent({
  components: {
    EventTableRow,
    EventTable,
  },
  props: {
    event: {
      type: Object as PropType<Sentry>,
      required: true,
    },
  },
  computed: {
    hasRequest() {
      return this.event.request !== undefined;
    },
    headers() {
      return this.event.request.headers || {};
    },
  },
});
</script>

<style lang="scss" scoped>
@import "assets/mixins";

.sentry-page-request {
}

.sentry-page-request__title {
  @include text-muted;
  @apply font-bold uppercase text-sm mb-5;
}

.sentry-page-request__title--sub {
  @apply mt-7;
}

.sentry-page-request__url {
  @apply mb-1 text-lg font-medium;
}
</style>
