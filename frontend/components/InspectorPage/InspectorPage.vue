<template>
  <div ref="main" class="inspector-page">
    <main class="inspector-page__in">
      <header class="inspector-page__header">
        <h2 class="inspector-page__header-title">
          {{ transaction.name }}
        </h2>
        <div class="inspector-page__header-meta">
          <span class="inspector-page__header-date">{{ date }}</span>
        </div>
      </header>

      <InspectorStatBoard :transaction="transaction" />
      <InspectorPageTimeline :event="event" />

      <section class="inspector-page__body">
        <h3 class="inspector-page__body-text">Url</h3>
        <EventTable class="inspector-page__body-table">
          <EventTableRow
            v-for="(value, name) in transaction.http.url"
            :key="name"
            :title="name"
          >
            {{ value }}
          </EventTableRow>
        </EventTable>
      </section>

      <section>
        <h3 class="inspector-page__body-text">Request</h3>
        <EventTable class="inspector-page__body-table">
          <EventTableRow
            v-for="(value, name) in transaction.http.request"
            :key="name"
            :title="name"
          >
            <template v-if="typeof value === 'string'">
              {{ value }}
            </template>
            <template v-else-if="!Array.isArray(value)">
              <EventTableRow v-for="(v, n) in value" :key="n" :title="n">
                {{ v }}
              </EventTableRow>
            </template>
          </EventTableRow>
        </EventTable>
      </section>
    </main>
  </div>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { InspectorTransaction, NormalizedEvent } from "~/config/types";
import moment from "moment/moment";
import InspectorPageTimeline from "~/components/InspectorPageTimeline/InspectorPageTimeline.vue";
import InspectorStatBoard from "~/components/InspectorStatBoard/InspectorStatBoard.vue";
import EventTable from "~/components/EventTable/EventTable.vue";
import EventTableRow from "~/components/EventTableRow/EventTableRow.vue";

export default defineComponent({
  components: {
    EventTableRow,
    EventTable,
    InspectorPageTimeline,
    InspectorStatBoard,
  },
  props: {
    event: {
      type: Object as PropType<NormalizedEvent>,
      required: true,
    },
  },
  computed: {
    transaction(): InspectorTransaction {
      return this.event?.payload[0] as unknown;
    },
    date() {
      return moment(this.event.timestamp).format("DD.MM.YYYY HH:mm:ss");
    },
  },
});
</script>

<style lang="scss" scoped>
@import "assets/mixins";

.inspector-page {
  @apply relative;
}

.inspector-page__in {
  @apply flex flex-col h-full flex-grow py-5 px-4 md:px-6 lg:px-8;
}

.inspector-page__header {
  @apply flex flex-col md:flex-row justify-between items-center mb-5;
}

.inspector-page__header-meta {
  @apply flex flex-col md:flex-row items-center gap-x-5;
}

.inspector-page__header-title {
  @apply text-sm sm:text-base md:text-lg lg:text-2xl;
}

.inspector-page__header-date {
  @include text-muted;
  @apply text-sm font-semibold;
}

.inspector-page__body {
  @apply py-5;
}

.inspector-page__body-text {
  @include text-muted;
  @apply font-bold uppercase text-sm mb-5;
}

.inspector-page__body-table {
  @apply mt-3;
}
</style>
