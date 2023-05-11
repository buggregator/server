<template>
  <div ref="main" class="http-dump-page">
    <main class="http-dump-page__main">
      <h2 class="text-2xl">
        <strong>{{ event.payload.request.method }}</strong>: <code>{{ uri }}</code>
      </h2>

      <section v-if="hasHeaders">
        <h1 class="mb-3 text-xl font-bold">Headers</h1>
        <EventTable>
          <EventTableRow :title="title" v-for="(value, title) in event.payload.request.headers">
            {{ value[0] || value }}
          </EventTableRow>
        </EventTable>
      </section>

      <section v-if="hasCookies">
        <h1 class="mb-3 text-xl font-bold">Cookie</h1>
        <EventTable>
          <EventTableRow :title="title" v-for="(value, title) in event.payload.request.cookies">
            {{ value }}
          </EventTableRow>
        </EventTable>
      </section>

      <section v-if="hasQuery">
        <h1 class="mb-3 text-xl font-bold">Query Parameters</h1>
        <EventTable>
          <EventTableRow :title="title" v-for="(value, title) in event.payload.request.query">
            {{ value }}
          </EventTableRow>
        </EventTable>
      </section>

      <section v-if="hasPostData">
        <h1 class="mb-3 text-xl font-bold">POST Data</h1>
        <EventTable>
          <EventTableRow :title="title" v-for="(value, title) in event.payload.request.post">
            {{ value }}
          </EventTableRow>
        </EventTable>
      </section>

      <section v-if="hasBody">
        <h1 class="mb-3 text-xl font-bold">Request Body</h1>
        <code class="border p-3 block">
          {{ event.payload.request.body }}
        </code>
      </section>
    </main>
  </div>
</template>

<script lang="ts">
import {defineComponent, PropType} from "vue";
import {NormalizedEvent} from "~/config/types";
import EventTable from "~/components/EventTable/EventTable.vue";
import EventTableRow from "~/components/EventTableRow/EventTableRow.vue";

export default defineComponent({
  components: {
    EventTable,
    EventTableRow
  },
  props: {
    event: {
      type: Object as PropType<NormalizedEvent>,
      required: true,
    },
  },
  data() {
    return {};
  },
  computed: {
    uri() {
      return decodeURI(this.event.payload.request.uri);
    },
    hasPostData(): boolean {
      return this.event.payload.request.post && Object.keys(this.event.payload.request.post).length > 0;
    },

    hasQuery(): boolean {
      return this.event.payload.request.query && Object.keys(this.event.payload.request.query).length > 0;
    },

    hasHeaders(): boolean {
      return Object.keys(this.event.payload.request.headers).length > 0;
    },

    hasCookies(): boolean {
      return Object.keys(this.event.payload.request.cookies).length > 0;
    },

    hasBody(): boolean {
      return this.event.payload.request.body && this.event.payload.request.body.length > 0;
    },
  },
});
</script>

<style lang="scss" scoped>
@import "assets/mixins";

.http-dump-page {
  @apply relative flex-1 flex flex-col;
}

.http-dump-page__main {
  @apply flex-1 flex flex-col h-full gap-y-5 py-5 px-4 md:px-6 lg:px-8;
}

</style>
