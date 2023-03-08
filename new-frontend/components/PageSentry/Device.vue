<template>
  <section v-if="hasDevice" class="sentry-device">
    <h3 class="sentry-section__title">device</h3>

    <TableC>
      <TableRow v-if="device.archs" title="Architectures">
        <CodeSnippet class="mt-3" language="json" :code="device.archs"/>
      </TableRow>

      <TableRow v-if="device.battery_level" title="Battery Level">
        {{ parseInt(device.battery_level) }}%
      </TableRow>

      <TableRow v-if="device.boot_time" title="Boot Time">
        {{ formatDate(device.boot_time) }}
      </TableRow>

      <TableRow v-if="device.brand" title="Brand">
        {{ device.brand }}
      </TableRow>

      <TableRow v-if="device.charging" title="Charging">
        {{ device.charging }}
      </TableRow>

      <TableRow v-if="device.family" title="Family">
        {{ device.family }}
      </TableRow>

      <TableRow v-if="device.free_memory" title="Free Memory">
        {{ humanFileSize(device.free_memory) }}
      </TableRow>

      <TableRow v-if="device.free_storage" title="Free Storage">
        {{ humanFileSize(device.free_storage) }}
      </TableRow>

      <TableRow v-if="device.id" title="Id">
        {{ device.id }}
      </TableRow>

      <TableRow v-if="device.language" title="Language">
        {{ device.language }}
      </TableRow>

      <TableRow v-if="device.low_memory" title="Low Memory">
        {{ device.low_memory }}
      </TableRow>

      <TableRow v-if="device.manufacturer" title="Manufacturer">
        {{ device.manufacturer }}
      </TableRow>

      <TableRow v-if="device.memory_size" title="Memory Size">
        {{ humanFileSize(device.memory_size) }}
      </TableRow>

      <TableRow v-if="device.model" title="Model">
        {{ device.model }}
      </TableRow>

      <TableRow v-if="device.model_id" title="Model Id">
        {{ device.model_id }}
      </TableRow>

      <TableRow v-if="device.name" title="Name">
        {{ device.name }}
      </TableRow>

      <TableRow v-if="device.orientation" title="Orientation">
        {{ device.orientation }}
      </TableRow>

      <TableRow v-if="device.screen_density" title="'Screen Density">
        {{ parseInt(device.screen_density) }}
      </TableRow>

      <TableRow v-if="device.screen_dpi" title="Screen DPI">
        {{ device.screen_dpi }}
      </TableRow>

      <TableRow v-if="device.screen_height_pixels" title="Screen Height Pixels">
        {{ device.screen_height_pixels }}
      </TableRow>

      <TableRow v-if="device.screen_width_pixels" title="Screen Width Pixels">
        {{ device.screen_width_pixels }}
      </TableRow>

      <TableRow v-if="device.simulator" title="Simulator">
        {{ device.simulator }}
      </TableRow>

      <TableRow v-if="device.storage_size" title="Storage Size">
        {{ humanFileSize(device.storage_size) }}
      </TableRow>

      <TableRow v-if="device.timezone" title="Timezone">
        {{ device.timezone }}
      </TableRow>

      <TableRow v-if="device.battery_temperature" title="Battery Temperature">
        {{ device.battery_temperature }}
      </TableRow>

      <TableRow v-if="device.locale" :title="'Locale'">
        {{ device.locale }}
      </TableRow>
    </TableC>
  </section>
</template>

<script lang="ts">
import {defineComponent, PropType} from "vue";
import {Sentry} from "~/config/types";
import TableC from "~/components/Table/Table.vue";
import TableRow from "~/components/Table/TableRow.vue";
import {humanFileSize} from "~/utils/formats";
import moment from "moment";

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
    hasDevice() {
      return this.event.contexts?.device !== undefined
    },
    device() {
      return this.event.contexts.device
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

.sentry-device {}

.sentry-section__title {
  @include text-muted;
  @apply font-bold uppercase text-sm mb-5;
}
</style>