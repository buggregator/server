<template>
  <div class="ray-type-notify">
    <File :file="file" />
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
    file(): unknown {
      return this.payload.origin;
    },
  },
});
</script>

<style lang="scss" scoped></style>
