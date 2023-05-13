<template>
  <div class="ray-type-model">
    <h3>
      Class name: <code>{{ payload.content.class_name }}</code>
    </h3>
    <ValueDump :value="payload.content.attributes"/>
  </div>
</template>

<script lang="ts">
import {defineComponent, PropType} from "vue";
import {RayPayload} from "~/config/types";
import ValueDump from "~/components/ValueDump/ValueDump.vue";

export default defineComponent({
  components: {ValueDump},
  props: {
    payload: {
      type: Object as PropType<RayPayload>,
      required: true,
    },
  },
});
</script>

<style lang="scss" scoped>
.ray-type-model {
}

.ray-type-model h3 {
  @apply p-3 border border-gray-300 dark:border-gray-600 bg-gray-800;
}

.ray-type-model code {
  @apply font-semibold;
}
</style>
