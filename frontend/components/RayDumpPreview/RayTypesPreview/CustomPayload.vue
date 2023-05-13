<template>
  <div class="ray-type-custom">
    <ValueDump v-if="!isEmptyValue" :value="eventValue" />
  </div>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { RayPayload } from "~/config/types";
import ValueDump from "~/components/ValueDump/ValueDump.vue";

export default defineComponent({
  components: { ValueDump },
  props: {
    payload: {
      type: Object as PropType<RayPayload>,
      required: true,
    },
  },
  computed: {
    isEmptyValue() {
      return this.eventValue === "";
    },
    eventValue(): unknown {
      return this.payload.content.content;
    },
  },
});
</script>

<style lang="scss" scoped>
.ray-type-log {
}
</style>
