<template>
  <section v-if="hasApp" class="sentry-page-app">
    <h3 class="sentry-page-app__title">app</h3>

    <EventTable>
      <EventTableRow v-if="app.type" title="App type">
        {{ app.type }}
      </EventTableRow>

      <EventTableRow v-if="app.app_build" title="App Build">
        {{ app.app_build }}
      </EventTableRow>

      <EventTableRow v-if="app.app_identifier" title="Build ID">
        {{ app.app_identifier }}
      </EventTableRow>

      <EventTableRow v-if="app.app_id" title="ID">
        {{ app.app_id }}
      </EventTableRow>

      <EventTableRow v-if="app.app_name" title="Build Name">
        {{ app.app_name }}
      </EventTableRow>

      <EventTableRow v-if="app.app_version" title="Version">
        {{ app.app_version }}
      </EventTableRow>

      <EventTableRow v-if="app.permissions" title="Permissions">
        <CodeSnippet class="mt-3" language="json" :code="app.permissions" />
      </EventTableRow>
    </EventTable>
  </section>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { Sentry } from "~/config/types";
import EventTableRow from "~/components/EventTableRow/EventTableRow.vue";
import EventTable from "~/components/EventTable/EventTable.vue";
import CodeSnippet from "~/components/CodeSnippet/CodeSnippet.vue";

export default defineComponent({
  components: {
    CodeSnippet,
    EventTable,
    EventTableRow,
  },
  props: {
    event: {
      type: Object as PropType<Sentry>,
      required: true,
    },
  },
  computed: {
    hasApp() {
      return this.event.contexts?.app !== undefined;
    },
    app() {
      return this.event.contexts.app;
    },
  },
});
</script>

<style lang="scss" scoped>
@import "assets/mixins";

.sentry-page-app {
  @apply py-5 px-4;
}

.sentry-page-app__title {
  @include text-muted;
  @apply font-bold uppercase text-sm mb-5;
}
</style>
