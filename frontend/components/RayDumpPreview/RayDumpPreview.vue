<template>
  <PreviewCard class="ray-dump-preview" :event="event">
    <RayDumpTypeMapper
      v-for="payload in event.payload.payload.payloads"
      :class="classes"
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
  computed: {
    classes() {
      let classes = [];

      classes.push(`text-${this.event.size}`);
      classes.push(`text-${this.event.color}-900 dark:text-${this.event.color}-200`);

      return classes
    },
  }
});
</script>
