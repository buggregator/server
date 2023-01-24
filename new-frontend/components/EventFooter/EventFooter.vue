<template>
  <div class="event-footer">
    <div class="event-footer__tags">
      <template v-if="originConfig">
        <span v-for="(value, key) in originConfig" :key="key">
          <strong>{{ key }}: </strong>{{ value }}
        </span>
      </template>
    </div>

    <div v-if="serverName" class="event-footer__host">
      <IconSvg name="host" class="event-footer__host-icon" />
      <span class="event-footer__host-name">
        {{ serverName }}
      </span>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import IconSvg from "~/components/IconSvg/IconSvg.vue";

interface OriginConfig {
  [key: string]: string;
}

export default defineComponent({
  components: {
    IconSvg,
  },
  props: {
    serverName: {
      type: String,
      default: "",
    },
    originConfig: {
      type: Object as PropType<OriginConfig>,
      default: null,
    },
  },
});
</script>

<style lang="scss" scoped>
.event-footer {
  @apply w-full flex justify-between;
}

.event-footer__tags {
  @apply flex flex-wrap items-center space-x-4 text-xs text-gray-400;
}

.event-footer__host {
  @apply inline-flex items-center justify-start space-x-2 py-1 text-gray-600 dark:text-gray-300;
}

.event-footer__host-icon {
  @apply w-4 h-4;
}

.event-footer__host-name {
  @apply flex-initial font-bold text-xs;
}
</style>
