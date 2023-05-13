<template>
  <div class="ray-type-measure">
    <h3 v-if="eventValue.is_new_timer" class="font-bold">
      Start measuring performance...
    </h3>

    <EventTable v-else>
      <EventTableRow title="Total time"> {{ totalTime }} s </EventTableRow>
      <EventTableRow title="Time since last call">
        {{ timeSinceLastCall }} s
      </EventTableRow>
      <EventTableRow title="Maximum memory usage">
        {{ maxMemoryUsage }}
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
    totalTime(): string {
      return this.convertMilliseconds(this.eventValue.total_time);
    },
    timeSinceLastCall(): string {
      return this.convertMilliseconds(this.eventValue.time_since_last_call);
    },
    maxMemoryUsage(): string {
      return this.prettySize(
        this.eventValue.max_memory_usage_during_total_time
      );
    },
  },
  methods: {
    convertMilliseconds(milliseconds: number): string {
      return (milliseconds / 1000).toFixed(4);
    },
    prettySize(bytes: number, separator = "", postFix = ""): string {
      if (bytes) {
        const sizes = ["Bytes", "KB", "MB", "GB", "TB"];
        const i = Math.min(
          parseInt(Math.floor(Math.log(bytes) / Math.log(1024)), 10),
          sizes.length - 1
        );
        return `${(bytes / 1024 ** i).toFixed(2)}${separator} ${
          sizes[i]
        }${postFix}`;
      }
      return "n/a";
    },
  },
});
</script>

<style lang="scss" scoped></style>
