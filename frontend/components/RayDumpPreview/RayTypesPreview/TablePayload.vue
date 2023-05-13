<template>
  <div class="ray-type-table">
    <EventTable>
      <EventTableRow
        v-for="(value, title) in eventValue.values"
        :key="title"
        :title="title"
      >
        <ValueDump :value="value" />
      </EventTableRow>
    </EventTable>
  </div>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { RayPayload } from "~/config/types";
import EventTable from "~/components/EventTable/EventTable.vue";
import EventTableRow from "~/components/EventTableRow/EventTableRow.vue";
import ValueDump from "~/components/ValueDump/ValueDump.vue";

export default defineComponent({
  components: {
    ValueDump,
    EventTableRow,
    EventTable,
  },
  props: {
    payload: {
      type: Object as PropType<RayPayload>,
      required: true,
    },
  },
  computed: {
    eventValue(): unknown {
      return this.payload.content;
    },
  },
});
</script>

<style lang="scss" scoped></style>
