<template>
  <div ref="main" class="http-dump-page">
    <main class="http-dump-page__main">
      <h2 class="http-dump-page__title">
        <span class="title-method">{{ event.payload.request.method }}</span>: <span class="title-uri">/{{ uri }}</span>
      </h2>

      <section v-if="hasHeaders" class="section-container">
        <h1>Headers</h1>
        <EventTable>
          <EventTableRow :title="title" v-for="(value, title) in event.payload.request.headers">
            {{ value[0] || value }}
          </EventTableRow>
        </EventTable>
      </section>

      <section v-if="hasCookies" class="section-container">
        <h1>Cookie</h1>
        <EventTable>
          <EventTableRow :title="title" v-for="(value, title) in event.payload.request.cookies">
            {{ value }}
          </EventTableRow>
        </EventTable>
      </section>

      <section v-if="hasQuery" class="section-container">
        <h1>Query Parameters</h1>
        <EventTable>
          <EventTableRow :title="title" v-for="(value, title) in event.payload.request.query">
            {{ value }}
          </EventTableRow>
        </EventTable>
      </section>

      <section v-if="hasPostData" class="section-container">
        <h1>POST Data</h1>
        <EventTable>
          <EventTableRow :title="title" v-for="(value, title) in event.payload.request.post">
            {{ value }}
          </EventTableRow>
        </EventTable>
      </section>

      <section v-if="hasAttachments" class="section-container">
        <h1>
          Attachments ({{ event.payload.request.files.length }})
        </h1>

        <div class="attachments">
          <Attachment
            v-for="a in event.payload.request.files"
            :key="a.id"
            :event="event"
            :attachment="a"
          />
        </div>
      </section>

      <section v-if="hasBody" class="section-container">
        <h1>Request Body</h1>
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
import Attachment from "~/components/Attachment/Attachment.vue";

export default defineComponent({
  components: {
    EventTable,
    EventTableRow,
    Attachment,
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

    hasAttachments(): boolean {
      return Object.keys(this.event.payload.request.files).length > 0;
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
  @apply flex-1 flex flex-col h-full gap-y-10 py-5 px-4 md:px-6 lg:px-8;
}

.http-dump-page__title {
  @apply text-2xl;
}

.title-method {
  @apply font-mono;
}

.title-uri {
  @apply font-mono font-bold;
}

.section-container {
  @apply flex-1;
}

.section-container h1 {
  @apply mb-3 text-xl font-bold;
}

.attachments {
  @apply flex gap-x-3;
}

</style>
