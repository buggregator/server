<template>
  <div class="ray-type-exception">
    <div class="ray-type-exception__wrap">
      <h3 class="mb-1">
        <code class="font-semibold">{{ eventValue.class }}</code>
      </h3>
      <div class="ray-type-exception__text">
        {{ eventValue.message }}
      </div>
    </div>
    <div class="ray-type-exception__files">
      <File
        v-for="(file, i) in eventValue.frames"
        :key="i"
        :file="file"
        :collapsed="i !== 0"
      />
    </div>
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
    eventValue() {
      return this.payload.content;
    },
  },
});
</script>

<style lang="scss" scoped></style>
