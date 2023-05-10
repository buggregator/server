<template>
  <PreviewCard class="var-dump-preview" :event="event">
    <ValueDump
     :value="eventValue"
     :type="event.payload.payload.type"
    />
  </PreviewCard>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { NormalizedEvent, VarDump } from "~/config/types";
import PreviewCard from "~/components/PreviewCard/PreviewCard.vue";
import ValueDump from "~/components/ValueDump/ValueDump.vue";

export default defineComponent({
  components: {
    PreviewCard,
    ValueDump,
  },
  props: {
    event: {
      type: Object as PropType<NormalizedEvent>,
      required: true,
    },
  },
  computed: {
    eventValue(): unknown {
      return (this.event.payload as VarDump).payload.value;
    },
  },
});
</script>

<style lang="scss" scoped>
.var-dump-preview {
  display: block;
}
</style>
