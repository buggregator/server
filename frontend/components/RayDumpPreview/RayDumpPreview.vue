<template>
  <PreviewCard class="ray-dump-preview" :event="event">
    <RayDumpTypeMapper
      v-for="payload in event.payload.payload.payloads"
      :payload="payload"
    />
  </PreviewCard>
</template>

<script lang="ts">
import {defineComponent, PropType} from "vue";
import {NormalizedEvent} from "~/config/types";
import RayDumpTypeMapper from "~/components/RayDumpTypeMapper/RayDumpTypeMapper.vue";
import PreviewCard from "~/components/PreviewCard/PreviewCard.vue";

export default defineComponent({
  components: {
    PreviewCard,
    RayDumpTypeMapper
  },
  props: {
    event: {
      type: Object as PropType<NormalizedEvent>,
      required: true,
    },
  },
});
</script>

<style lang="scss" scoped>

</style>
