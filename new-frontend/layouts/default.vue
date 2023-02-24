<template>
  <div class="main-layout">
    <div class="main-layout__sidebar-wrap">
      <layout-sidebar :is-connected="isConnected" />
    </div>

    <div class="main-layout__content">
      <slot />
    </div>
  </div>
</template>

<script lang="ts">
import LayoutSidebar from "~/layouts/LayoutSidebar/LayoutSidebar.vue";
import { defineComponent } from "vue";

export default defineComponent({
  components: {
    LayoutSidebar,
  },
  computed: {
    isConnected() {
      // return this.$store.getters['ws/connected']
      return false;
    },
  },
});
</script>

<style lang="scss" scoped>
.main-layout {
  @apply flex h-screen items-stretch;
}

.main-layout__sidebar-wrap {
  @apply w-10 md:w-14 lg:w-16 flex-none border-r;
}

.main-layout__content {
  @apply flex flex-col h-full flex-1;

  & > div {
    @apply flex flex-col h-full flex-1;
  }
}

.main-layout__sidebar {
  @apply w-10 md:w-14 lg:w-16 fixed h-screen;
}
</style>
