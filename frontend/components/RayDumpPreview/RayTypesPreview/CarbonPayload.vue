<template>
  <div class="ray-type-carbon">
    <EventTable>
      <EventTableRow title="Formatted">
        {{ eventValue.formatted }}
      </EventTableRow>
      <EventTableRow title="Timezone">
        {{ eventValue.timezone }}
      </EventTableRow>
      <EventTableRow title="Timestamp">
        {{ eventValue.timestamp }}
      </EventTableRow>
    </EventTable>
  </div>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { RayPayload } from "~/config/types";
import EventTable from "~/components/EventTable/EventTable.vue";
import EventTableRow from "~/components/EventTableRow/EventTableRow.vue";

export default defineComponent({
  components: {
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
