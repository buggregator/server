<template>
  <div class="ray-type-trace">
    <File v-for="(file, i) in frames" :key="i" :file="file" />
  </div>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { RayPayload } from "~/config/types";
import File from "~/components/FileView/FileView.vue";

export default defineComponent({
  components: { File },
  props: {
    payload: {
      type: Object as PropType<RayPayload>,
      required: true,
    },
  },
  computed: {
    frames(): unknown {
      return this.payload.content!.frames;
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
