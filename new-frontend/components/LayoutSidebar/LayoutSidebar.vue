<template>
  <aside class="sidebar">
    <nav class="sidebar__nav">
      <NuxtLink to="/" title="Events" class="sidebar__link">
        <icon-svg class="sidebar__link-icon" name="events" />
      </NuxtLink>

      <NuxtLink to="/sentry" title="Sentry logs" class="sidebar__link">
        <icon-svg class="sidebar__link-icon" name="sentry" />
      </NuxtLink>

      <NuxtLink to="/profiler" title="Profiler" class="sidebar__link">
        <icon-svg class="sidebar__link-icon" name="profiler" />
      </NuxtLink>

      <NuxtLink to="/smtp" title="SMTP mails" class="sidebar__link">
        <icon-svg class="sidebar__link-icon" name="smtp" />
      </NuxtLink>

      <NuxtLink to="/inspector" title="Inspector logs" class="sidebar__link">
        <icon-svg class="sidebar__link-icon" name="inspector" />
      </NuxtLink>

      <NuxtLink to="/settings" title="Settings" class="sidebar__link">
        <icon-svg class="sidebar__link-icon" name="settings" />
      </NuxtLink>
    </nav>

    <div class="sidebar__info">
      <div class="sidebar__info-item" :title="title">
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
.sidebar {
  @apply bg-gray-200 dark:bg-gray-800 border-gray-300 dark:border-gray-500 flex flex-col justify-between z-50;
}

.sidebar__in {
  @apply divide-y divide-gray-300 dark:divide-gray-600;
}

.sidebar__supIcon {
  @apply absolute top-2 w-2 h-2 bg-red-600 right-2 rounded-full transition transition-all duration-300;
}

.sidebar__link {
  @apply text-blue-500 p-3 md:p-4 lg:p-5 block hover:bg-blue-500 hover:text-white relative;

  &.router-link-active {
    @apply bg-blue-700 text-blue-200;
  }
}

.sidebar__link-icon {
  @apply fill-current;
}

.sidebar__info {
  @apply divide-y divide-gray-300;
}

.sidebar__info-item {
  @apply p-2 md:p-3 lg:p-4;
}
</style>
