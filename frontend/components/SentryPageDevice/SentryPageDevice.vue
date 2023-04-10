<template>
  <section v-if="hasDevice" class="sentry-page-device">
    <h3 class="sentry-page-device__title">device</h3>

    <EventTable>
      <EventTableRow v-if="device.archs" title="Architectures">
        <CodeSnippet class="mt-3" language="json" :code="device.archs" />
      </EventTableRow>

      <EventTableRow v-if="device.battery_level" title="Battery Level">
        {{ parseInt(device.battery_level) }}%
      </EventTableRow>

      <EventTableRow v-if="device.boot_time" title="Boot Time">
        {{ formatDate(device.boot_time) }}
      </EventTableRow>

      <EventTableRow v-if="device.brand" title="Brand">
        {{ device.brand }}
      </EventTableRow>

      <EventTableRow v-if="device.charging" title="Charging">
        {{ device.charging }}
      </EventTableRow>

      <EventTableRow v-if="device.family" title="Family">
        {{ device.family }}
      </EventTableRow>

      <EventTableRow v-if="device.free_memory" title="Free Memory">
        {{ humanFileSize(device.free_memory) }}
      </EventTableRow>

      <EventTableRow v-if="device.free_storage" title="Free Storage">
        {{ humanFileSize(device.free_storage) }}
      </EventTableRow>

      <EventTableRow v-if="device.id" title="Id">
        {{ device.id }}
      </EventTableRow>

      <EventTableRow v-if="device.language" title="Language">
        {{ device.language }}
      </EventTableRow>

      <EventTableRow v-if="device.low_memory" title="Low Memory">
        {{ device.low_memory }}
      </EventTableRow>

      <EventTableRow v-if="device.manufacturer" title="Manufacturer">
        {{ device.manufacturer }}
      </EventTableRow>

      <EventTableRow v-if="device.memory_size" title="Memory Size">
        {{ humanFileSize(device.memory_size) }}
      </EventTableRow>

      <EventTableRow v-if="device.model" title="Model">
        {{ device.model }}
      </EventTableRow>

      <EventTableRow v-if="device.model_id" title="Model Id">
        {{ device.model_id }}
      </EventTableRow>

      <EventTableRow v-if="device.name" title="Name">
        {{ device.name }}
      </EventTableRow>

      <EventTableRow v-if="device.orientation" title="Orientation">
        {{ device.orientation }}
      </EventTableRow>

      <EventTableRow v-if="device.screen_density" title="Screen Density">
        {{ parseInt(device.screen_density) }}
      </EventTableRow>

      <EventTableRow v-if="device.screen_dpi" title="Screen DPI">
        {{ device.screen_dpi }}
      </EventTableRow>

      <EventTableRow
        v-if="device.screen_height_pixels"
        title="Screen Height Pixels"
      >
        {{ device.screen_height_pixels }}
      </EventTableRow>

      <EventTableRow
        v-if="device.screen_width_pixels"
        title="Screen Width Pixels"
      >
        {{ device.screen_width_pixels }}
      </EventTableRow>

      <EventTableRow v-if="device.simulator" title="Simulator">
        {{ device.simulator }}
      </EventTableRow>

      <EventTableRow v-if="device.storage_size" title="Storage Size">
        {{ humanFileSize(device.storage_size) }}
      </EventTableRow>

      <EventTableRow v-if="device.timezone" title="Timezone">
        {{ device.timezone }}
      </EventTableRow>

      <EventTableRow
        v-if="device.battery_temperature"
        title="Battery Temperature"
      >
        {{ device.battery_temperature }}
      </EventTableRow>

      <EventTableRow v-if="device.locale" :title="'Locale'">
        {{ device.locale }}
      </EventTableRow>
    </EventTable>
  </section>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { Sentry } from "~/config/types";
import CodeSnippet from "~/components/CodeSnippet/CodeSnippet.vue";
import { humanFileSize } from "~/utils/formats";
import moment from "moment";
import EventTable from "~/components/EventTable/EventTable.vue";
import EventTableRow from "~/components/EventTableRow/EventTableRow.vue";

export default defineComponent({
  components: {
    EventTableRow,
    EventTable,
    CodeSnippet,
  },
  props: {
    event: {
      type: Object as PropType<Sentry>,
      required: true,
    },
  },
  computed: {
    hasDevice() {
      return this.event.contexts?.device !== undefined;
    },
    device() {
      return this.event.contexts.device;
    },
  },
  methods: {
    humanFileSize,
    formatDate(date: string) {
      return moment(date).toLocaleString();
    },
  },
});
</script>

<style lang="scss" scoped>
@import "assets/mixins";

.sentry-page-device {
}

.sentry-page-device__title {
  @include text-muted;
  @apply font-bold uppercase text-sm mb-5;
}
</style>
