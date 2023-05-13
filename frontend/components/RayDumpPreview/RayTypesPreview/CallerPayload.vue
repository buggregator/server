<template>
  <div class="ray-type-caller">
    <h3>Called from</h3>
    <a
      class="ray-type-caller__name"
      :href="`phpstorm://open?file=${encodeURIComponent(file.file_name)}&line=${
        file.line_number
      }`"
    >
      <code class="ray-type-caller__code"
        >{{ file.class || "null" }}:{{ file.method }}</code
      >
    </a>
  </div>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { RayPayload } from "~/config/types";

export default defineComponent({
  props: {
    payload: {
      type: Object as PropType<RayPayload>,
      required: true,
    },
  },
  computed: {
    file() {
      return this.payload.content.frame;
    },
  },
});
</script>

<style lang="scss" scoped>
.dark .ray-type-caller__name {
  --tw-text-opacity: 1;
  color: rgba(219, 234, 254, var(--tw-text-opacity));
}
.ray-type-caller {
  &__name {
    --tw-text-opacity: 1;
    color: rgba(96, 165, 250, var(--tw-text-opacity));
    text-decoration: underline;
  }
  &__code {
    word-break: break-all;
    font-weight: 600;
  }
}
</style>
