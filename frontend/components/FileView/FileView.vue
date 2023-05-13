<template>
  <div class="event-ray-file" @click="collapsed = !collapsed">
    <div class="event-ray-file__header">
      <div :title="file.file_name" class="event-ray-file__title">
        <div>
          {{ file.class || "null" }}:{{ file.method }}
          <span class="text-muted">at line</span>
          {{ file.line_number }}
        </div>
        <span class="text-muted">{{ file.file_name }}</span>
      </div>
      <div v-if="hasSnippet" class="event-ray-file__icon">
        <svg
          viewBox="0 0 16 16"
          fill="currentColor"
          height="100%"
          width="100%"
          :class="{ 'transform rotate-180': collapsed }"
        >
          <path
            d="M14,11.75a.74.74,0,0,1-.53-.22L8,6.06,2.53,11.53a.75.75,0,0,1-1.06-1.06l6-6a.75.75,0,0,1,1.06,0l6,6a.75.75,0,0,1,0,1.06A.74.74,0,0,1,14,11.75Z"
          ></path>
        </svg>
      </div>
    </div>

    <div v-if="hasSnippet && !collapsed" class="event-ray-file__body">
      <div
        v-for="(line, i) in file.snippet"
        :key="i"
        class="event-ray-file__snippet"
        :class="{
          'bg-pink-800 text-white': file.line_number == line.line_number,
        }"
      >
        <div class="w-12">{{ line.line_number }}.</div>
        <pre>{{ line.text }}</pre>
      </div>
    </div>
  </div>
</template>
<script lang="ts">
import { defineComponent } from "vue";

export default defineComponent({
  props: {
    file: {
      file_name: String,
      line_number: Number,
      class: String,
      method: String,
      vendor_frame: Boolean,
    },
  },
  data() {
    return {
      collapsed: true,
    };
  },
  computed: {
    hasSnippet() {
      return this.file!.snippet && this.file!.snippet.length > 0;
    },
  },
});
</script>
<style lang="scss">
.event-ray-file {
  @apply text-xs cursor-pointer border-b border-purple-200 dark:border-gray-600;

  &__header {
    @apply bg-purple-50 dark:bg-gray-800 py-2 px-3 flex space-x-2 justify-between items-start;
  }

  &__title {
    @apply break-all font-semibold;
  }

  &__icon {
    @apply w-5 h-4 border border-purple-300 shadow bg-white dark:bg-gray-600 py-1 rounded;
  }

  &__body {
    @apply bg-gray-900 p-2 overflow-x-scroll;
  }

  &__snippet {
    @apply flex text-gray-100;
  }
}
</style>
