<template>
  <div class="ray-type-query">
    <CodeSnippet language="sql" class="event-ray__query-snippet" :code="formattedSql" />
    <EventTable class="event-ray__query-table">
      <EventTableRow title="Connection name">
        {{ payload.content.connection_name }}
      </EventTableRow>
      <EventTableRow title="Time">
        {{ payload.content.time }}ms
      </EventTableRow>
    </EventTable>
  </div>
</template>

<script lang="ts">
import {defineComponent, PropType} from "vue";
import {RayPayload} from "~/config/types";
import EventTable from "~/components/EventTable/EventTable.vue";
import EventTableRow from "~/components/EventTableRow/EventTableRow.vue";
import CodeSnippet from "~/components/CodeSnippet/CodeSnippet.vue";

export default defineComponent({
  components: {
    EventTableRow,
    EventTable,
    CodeSnippet
  },
  props: {
    payload: {
      type: Object as PropType<RayPayload>,
      required: true,
    },
  },
  computed: {
    formattedSql() {
      return this.payload.content.bindings.reduce((sql, currentValue) => {
        return sql.replace(/\?/, `'${currentValue}'`)
      }, this.payload.content.sql)
    }
  },
});
</script>

<style lang="scss" scoped></style>
