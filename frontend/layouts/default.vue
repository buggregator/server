<template>
  <div class="main-layout">
    <div class="main-layout__sidebar-wrap">
      <LayoutSidebar :is-connected="isConnected" />
    </div>

    <div class="main-layout__content">
      <slot />
    </div>
  </div>
</template>

<script lang="ts">
import LayoutSidebar from "~/components/LayoutSidebar/LayoutSidebar.vue";
import { defineComponent } from "vue";
import { THEME_MODES, useThemeStore } from "~/stores/theme";
import { storeToRefs } from "pinia";
import { useNuxtApp } from "#app";

export default defineComponent({
  components: {
    LayoutSidebar,
  },

  setup() {
    const themeStore = useThemeStore();
    const { themeType } = storeToRefs(themeStore);

    if (process.client) {
      const { $events } = useNuxtApp();

      if (!$events?.items?.length) {
        $events.getAll();
      }
    }

    return {
      themeType,
    };
  },
  computed: {
    isConnected() {
      // return this.$store.getters['ws/connected']
      return false;
    },
  },
  mounted() {
    if (this.themeType === THEME_MODES.DARK) {
      document?.documentElement?.classList?.add(THEME_MODES.DARK);
    } else {
      document?.documentElement?.classList?.remove(THEME_MODES.DARK);
    }
  },
});
</script>

<style lang="scss" scoped>
.main-layout {
  @apply flex min-h-screen items-stretch relative;
}

.main-layout__sidebar-wrap {
  @apply w-10 md:w-14 lg:w-16 flex-none border-r border-gray-200 dark:border-gray-700;
}

.main-layout__content {
  @apply flex flex-col h-full flex-1 w-full min-h-screen;

  & > div {
    @apply flex flex-col h-full flex-1;
  }
}

.main-layout__sidebar {
  @apply w-10 md:w-14 lg:w-16 fixed h-screen;
}
</style>
