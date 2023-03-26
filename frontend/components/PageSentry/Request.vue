<template>
  <section v-if="hasRequest" class="sentry-request">
    <h3 class="sentry-section__title">request</h3>

    <h3 class="sentry-request__url">
      <strong>{{ event.request.method }}:</strong> {{ event.request.url }}
    </h3>


    <h3 class="sentry-section__title sentry-request__subtitle">headers</h3>
    <TableC>
      <TableRow v-for="(value, title) in headers" :key="title" :title="title">
        {{ value[0] || value}}
      </TableRow>
    </TableC>
  </section>
</template>

<script lang="ts">
import {defineComponent, PropType} from "vue";
import {Sentry} from "~/config/types";
import TableC from "~/components/Table/Table.vue";
import TableRow from "~/components/Table/TableRow.vue";

export default defineComponent({
  components: {
    TableC,
    TableRow,
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

.sentry-request {
}

.sentry-request__subtitle,
.sentry-section__title {
  @include text-muted;
  @apply font-bold uppercase text-sm mb-5;
}

.sentry-request__subtitle {
  @apply mt-7;
}

.sentry-request__url {
  @apply mb-1 text-lg font-medium;
}
</style>