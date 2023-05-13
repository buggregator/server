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

<style lang="scss" scoped></style>
