<template>
  <a :href="downloadUrl" target="_blank" class="smtp-attachment">
    <svg
      xmlns="http://www.w3.org/2000/svg"
      fill="currentColor"
      viewBox="0 0 50 50"
      width="25px"
      height="25px"
    >
      <path
        d="M 7 2 L 7 48 L 43 48 L 43 14.59375 L 42.71875 14.28125 L 30.71875 2.28125 L 30.40625 2 Z M 9 4 L 29 4 L 29 16 L 41 16 L 41 46 L 9 46 Z M 31 5.4375 L 39.5625 14 L 31 14 Z"
      />
    </svg>
    <div class="smtp-attachment--meta">
      <div class="smtp-attachment--name">
        {{ attachment.name }}
      </div>
      <div class="smtp-attachment--size">({{ size }})</div>
    </div>
  </a>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { NormalizedEvent, SmtpAttachment } from "~/config/types";
import { REST_API_URL } from "~/utils/events-transport";
import { humanFileSize } from "~/utils/formats";

export default defineComponent({
  props: {
    event: {
      type: Object as PropType<NormalizedEvent>,
      required: true,
    },
    attachment: {
      type: Object as PropType<SmtpAttachment>,
      required: true,
    },
  },
  computed: {
    downloadUrl(): string {
      return `${REST_API_URL}/api/smtp/${this.event.id}/attachment/${this.attachment.id}`;
    },
    size(): string {
      return humanFileSize(this.attachment.size);
    },
  },
});
</script>

<style lang="scss" scoped>
@import "assets/mixins";

.smtp-attachment {
  @apply border border-gray-300 px-3 py-2 flex items-center;

  > svg {
    @apply mr-3;
  }
}

.smtp-attachment--meta {
  @apple flex flex-col justify-start;
}

.smtp-attachment--name {
  @apply font-bold text-xs;
}

.smtp-attachment--size {
  @apply text-xs;
}
</style>
