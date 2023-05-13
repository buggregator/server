<template>
  <div class="ray-type-job">
    <ValueDump :value="payload.content.job" />
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
});
</script>

<style lang="scss" scoped></style>
