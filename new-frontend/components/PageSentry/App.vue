<template>
  <section v-if="hasApp" class="sentry-app">
    <h3 class="sentry-section__title">app</h3>

    <TableC>
      <TableRow v-if="app.type" title="App type">
        {{ app.type }}
      </TableRow>

      <TableRow v-if="app.app_build" title="App Build">
        {{ app.app_build }}
      </TableRow>

      <TableRow v-if="app.app_identifier" title="Build ID">
        {{ app.app_identifier }}
      </TableRow>

      <TableRow v-if="app.app_id" title="ID">
        {{ app.app_id }}
      </TableRow>

      <TableRow v-if="app.app_name" title="Build Name">
        {{ app.app_name }}
      </TableRow>

      <TableRow v-if="app.app_version" title="Version">
        {{ app.app_version }}
      </TableRow>

      <TableRow v-if="app.permissions" title="Permissions">
        <CodeSnippet class="mt-3" language="json" :code="app.permissions"/>
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
    hasApp() {
      return this.event.contexts?.app !== undefined
    },
    app() {
      return this.event.contexts.app
    },
  },
});
</script>

<style lang="scss" scoped>
@import "assets/mixins";

.sentry-app {
  @apply py-5 px-4;
}

.sentry-section__title {
  @include text-muted;
  @apply font-bold uppercase text-sm mb-5;
}
</style>