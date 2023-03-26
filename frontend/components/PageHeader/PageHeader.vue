<template>
  <header class="page-header">
    <div class="page-header__title"><slot />&nbsp;</div>

    <button
      v-if="buttonTitle"
      class="page-header__btn-clear"
      @click="buttonClick"
    >
      {{ buttonTitle }}
    </button>
  </header>
</template>

<script lang="ts">
import { defineComponent } from "vue";

export default defineComponent({
  props: {
    title: {
      type: String,
      default: "",
    },
    buttonTitle: {
      type: String,
      default: "",
    },
  },
  emits: {
    delete(payload: boolean) {
      return payload;
    },
  },
  methods: {
    buttonClick() {
      this.$emit("delete", true);
    },
  },
});
</script>

<style lang="scss" scoped>
.page-header {
  @apply p-3 flex justify-between border-b border-gray-600 text-2xl font-bold;
}

.page-header__title {
}

.page-header__btn-clear {
  @apply px-3 py-1 text-xs bg-red-800 text-white rounded-sm hover:bg-red-700 transition transition-all duration-300;
}
</style>
