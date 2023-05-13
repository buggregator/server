<template>
  <div class="ray-type-event">
    <ValueDump v-if="payload.content.class_based_event" :value="payload.content.event"/>
    <h3>
      Event: <code>{{ payload.content.name }}</code>
    </h3>
    <ValueDump v-if="payload.content.payload" :value="payload.content.payload"/>
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
.ray-type-event {
}

.ray-type-event h3 {
  @apply p-3 border border-gray-300 dark:border-gray-600 bg-gray-800;
}

.ray-type-event code {
  @apply font-semibold;
}
</style>
