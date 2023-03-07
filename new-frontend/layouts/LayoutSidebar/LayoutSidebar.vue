<template>
  <aside class="layout-sidebar">
    <nav class="layout-sidebar__nav">
      <NuxtLink to="/" title="Events" class="layout-sidebar__link">
        <icon-svg class="layout-sidebar__link-icon" name="events" />
      </NuxtLink>

      <NuxtLink to="/sentry" title="Sentry logs" class="layout-sidebar__link">
        <icon-svg class="layout-sidebar__link-icon" name="sentry" />
      </NuxtLink>

      <NuxtLink to="/profiler" title="Profiler" class="layout-sidebar__link">
        <icon-svg class="layout-sidebar__link-icon" name="profiler" />
      </NuxtLink>

      <NuxtLink to="/smtp" title="SMTP mails" class="layout-sidebar__link">
        <icon-svg class="layout-sidebar__link-icon" name="smtp" />
      </NuxtLink>

      <NuxtLink
        to="/inspector"
        title="Inspector logs"
        class="layout-sidebar__link"
      >
        <icon-svg class="layout-sidebar__link-icon" name="inspector" />
      </NuxtLink>

      <NuxtLink to="/settings" title="Settings" class="layout-sidebar__link">
        <icon-svg class="layout-sidebar__link-icon" name="settings" />
      </NuxtLink>
    </nav>

    <div class="layout-sidebar__info">
      <div class="layout-sidebar__info-item" :title="title">
        <icon-svg v-if="isConnected" name="connected" />
        <icon-svg v-if="!isConnected" name="disconnected" />
      </div>
    </div>
  </aside>
</template>

<script lang="ts">
import IconSvg from "~/components/IconSvg/IconSvg.vue";
import { defineComponent } from "vue";

export default defineComponent({
  components: { IconSvg },
  props: {
    isConnected: {
      type: Boolean,
      required: true,
    },
  },
  computed: {
    title(): string {
      return this.isConnected
        ? "Websocket connected"
        : "Websocket disconnected";
    },
  },
});
</script>

<style lang="scss" scoped>
.layout-sidebar {
  @apply bg-gray-200 dark:bg-gray-800 border-gray-300 dark:border-gray-500 flex flex-col justify-between z-50 w-full h-full;
}

.layout-sidebar__in {
  @apply divide-y divide-gray-300 dark:divide-gray-600;
}

.layout-sidebar__supIcon {
  @apply absolute top-2 w-2 h-2 bg-red-600 right-2 rounded-full transition transition-all duration-300;
}

.layout-sidebar__link {
  @apply text-blue-500 p-3 md:p-4 lg:p-5 block hover:bg-blue-500 hover:text-white relative;

  &.router-link-active {
    @apply bg-blue-700 text-blue-200;
  }
}

.layout-sidebar__link-icon {
  @apply fill-current;
}

.layout-sidebar__info {
  @apply divide-y divide-gray-300;
}

.layout-sidebar__info-item {
  @apply p-2 md:p-3 lg:p-4;
}
</style>
