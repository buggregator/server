<template>
  <div class="ray-type-view">
    <h3>
      View: <code>{{ payload.content.view_path_relative_to_project_root }}</code>
    </h3>
    <ValueDump :value="payload.content.data"/>
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
.ray-type-view {
}

.ray-type-view h3 {
  @apply p-3 border border-gray-300 dark:border-gray-600 bg-gray-800;
}

.ray-type-view code {
  @apply font-semibold;
}
</style>
