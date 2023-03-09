<template>
  <div ref="main" class="page-inspector">
    <main class="page-inspector__main">
      <header class="page-inspector__header">
        <h2 class="page-inspector__header-title">
          {{ transaction.name }}
        </h2>
        <div class="page-inspector__header-meta">
          <button class="page-inspector__delete-button" @click="$emit('delete', event)">
            <IconSvg name="trash-bin"/>
          </button>

          <span class="page-inspector__header-date">{{ date }}</span>
        </div>
      </header>

      <StatBoard :transaction="transaction" />
      <Timeline :event="event"/>

      <section class="py-5">
        <h3 class="text-muted font-bold uppercase text-sm mb-5">Url</h3>
        <TableC class="mt-3">
          <TableRow v-for="(value, name) in transaction.http.url" :key="name" :title="name">
            {{ value }}
          </TableRow>
        </TableC>
      </section>

      <section class="inspector-request">
        <h3 class="text-muted font-bold uppercase text-sm mb-5">Request</h3>
        <TableC class="mt-3">
          <TableRow v-for="(value, name) in transaction.http.request" :key="name" :title="name">
            <template v-if="typeof value==='string'">
              {{ value }}
            </template>
            <template v-else-if="!Array.isArray(value)">
              <TableRow v-for="(v, n) in value" :key="n" :title="n">
                {{ v }}
              </TableRow>
            </template>
          </TableRow>
        </TableC>
      </section>
    </main>
  </div>
</template>

<script lang="ts">
import {defineComponent, PropType} from "vue";
import {InspectorTransaction, NormalizedEvent} from "~/config/types";
import IconSvg from "~/components/IconSvg/IconSvg.vue";
import moment from "moment/moment";
import TableC from "~/components/Table/Table.vue";
import TableRow from "~/components/Table/TableRow.vue";
import Timeline from "./Timeline.vue";
import StatBoard from "./StatBoard.vue";

export default defineComponent({
  components: {
    Timeline,
    StatBoard,
    IconSvg,
    TableC,
    TableRow,
  },
  props: {
    event: {
      type: Object as PropType<NormalizedEvent>,
      required: true,
    },
  },
  computed: {
    transaction(): InspectorTransaction {
      return this.event.payload[0];
    },
    date() {
      return moment(this.event.timestamp).format('DD.MM.YYYY HH:mm:ss')
    },
  }
})
</script>

<style lang="scss" scoped>
@import "assets/mixins";

.page-inspector {
  @apply relative;
}

.page-inspector__main {
  @apply flex flex-col h-full flex-grow py-5 px-4 md:px-6 lg:px-8;
}

.page-inspector__header {
  @apply flex flex-col md:flex-row justify-between items-center mb-5;
}

.page-inspector__header-meta {
  @apply flex flex-col md:flex-row items-center gap-x-5;
}

.page-inspector__header-title {
  @apply text-sm sm:text-base md:text-lg lg:text-2xl;
}

.page-inspector__header-date {
  @include text-muted;
  @apply text-sm font-semibold;
}
</style>