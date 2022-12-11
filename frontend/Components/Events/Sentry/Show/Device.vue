<template>
  <section class="py-5 px-4 md:px-6 lg:px-8 border-b" v-if="event.contexts.device">
    <h3 class="text-muted font-bold uppercase text-sm mb-5">device</h3>

    <Table>
      <TableRow v-if="event.contexts.device.archs" :title="'Architectures'">
        <CodeSnippet class="mt-3" language="json" :code="event.contexts.device.archs"/>
      </TableRow>

      <TableRow v-if="event.contexts.device.battery_level" :title="'Battery Level'">
        {{ parseInt(event.contexts.device.battery_level) }}%
      </TableRow>

      <TableRow v-if="event.contexts.device.boot_time" :title="'Boot Time'">
        {{ format_date(event.contexts.device.boot_time) }}
      </TableRow>

      <TableRow v-if="event.contexts.device.brand" :title="'Brand'">
        {{ event.contexts.device.brand }}
      </TableRow>

      <TableRow v-if="event.contexts.device.charging" :title="'Charging'">
        {{ event.contexts.device.charging }}
      </TableRow>

      <TableRow v-if="event.contexts.device.family" :title="'Family'">
        {{ event.contexts.device.family }}
      </TableRow>

      <TableRow v-if="event.contexts.device.free_memory" :title="'Free Memory'">
        {{ prettyBytes(event.contexts.device.free_memory, 2, true) }}
      </TableRow>

      <TableRow v-if="event.contexts.device.free_storage" :title="'Free Storage'">
        {{ prettyBytes(event.contexts.device.free_storage, 2, true) }}
      </TableRow>

      <TableRow v-if="event.contexts.device.id" :title="'Id'">
        {{ event.contexts.device.id }}
      </TableRow>

      <TableRow v-if="event.contexts.device.language" :title="'Language'">
        {{ event.contexts.device.language }}
      </TableRow>

      <TableRow v-if="event.contexts.device.low_memory" :title="'Low Memory'">
        {{ event.contexts.device.low_memory }}
      </TableRow>

      <TableRow v-if="event.contexts.device.manufacturer" :title="'Manufacturer'">
        {{ event.contexts.device.manufacturer }}
      </TableRow>

      <TableRow v-if="event.contexts.device.memory_size" :title="'Memory Size'">
        {{ prettyBytes(event.contexts.device.memory_size, 2, true) }}
      </TableRow>

      <TableRow v-if="event.contexts.device.model" :title="'Model'">
        {{ event.contexts.device.model }}
      </TableRow>

      <TableRow v-if="event.contexts.device.model_id" :title="'Model Id'">
        {{ event.contexts.device.model_id }}
      </TableRow>

      <TableRow v-if="event.contexts.device.name" :title="'Name'">
        {{ event.contexts.device.name }}
      </TableRow>

      <TableRow v-if="event.contexts.device.orientation" :title="'Orientation'">
        {{ event.contexts.device.orientation }}
      </TableRow>

      <TableRow v-if="event.contexts.device.screen_density" :title="'Screen Density'">
        {{ parseInt(event.contexts.device.screen_density) }}
      </TableRow>

      <TableRow v-if="event.contexts.device.screen_dpi" :title="'Screen DPI'">
        {{ event.contexts.device.screen_dpi }}
      </TableRow>

      <TableRow v-if="event.contexts.device.screen_height_pixels" :title="'Screen Height Pixels'">
        {{ event.contexts.device.screen_height_pixels }}
      </TableRow>

      <TableRow v-if="event.contexts.device.screen_width_pixels" :title="'Screen Width Pixels'">
        {{ event.contexts.device.screen_width_pixels }}
      </TableRow>

      <TableRow v-if="event.contexts.device.simulator" :title="'Simulator'">
        {{ event.contexts.device.simulator }}
      </TableRow>

      <TableRow v-if="event.contexts.device.storage_size" :title="'Storage Size'">
        {{ prettyBytes(event.contexts.device.storage_size, 2, true) }}
      </TableRow>

      <TableRow v-if="event.contexts.device.timezone" :title="'Timezone'">
        {{ event.contexts.device.timezone }}
      </TableRow>

      <TableRow v-if="event.contexts.device.battery_temperature" :title="'Battery Temperature'">
        {{ event.contexts.device.battery_temperature }}
      </TableRow>

      <TableRow v-if="event.contexts.device.locale" :title="'Locale'">
        {{ event.contexts.device.locale }}
      </TableRow>
    </Table>
  </section>
</template>

<script>
import Table from "@/Components/UI/Table";
import TableRow from "@/Components/UI/TableRow";
import CodeSnippet from "@/Components/UI/CodeSnippet";

export default {
  components: {
    Table, TableRow, CodeSnippet
  },
  props: {
    event: Object
  },
  methods: {
    format_date(value) {
      if (value) {
        return new Date(parseInt(value)).toString()
      }
    },
    prettyBytes: function (bytes, decimals, kib, maxunit) {
      kib = kib || false
      if (bytes === 0) return '0 Bytes'
      if (isNaN(parseFloat(bytes)) && !isFinite(bytes)) return ''
      const k = kib ? 1024 : 1000
      const dm = decimals != null && !isNaN(decimals) && decimals >= 0 ? decimals : 2
      const sizes = kib ? ['Bytes', 'KiB', 'MiB', 'GiB', 'TiB', 'PiB', 'EiB', 'ZiB', 'YiB', 'BiB'] : ['Bytes', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB', 'BB']
      let i = Math.floor(Math.log(bytes) / Math.log(k));
      if (maxunit !== undefined) {
        const index = sizes.indexOf(maxunit)
        if (index !== -1) i = index
      }
      return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i]
    }
  }
}
</script>
