<template>
  <div
    :id="event.id"
    ref="event"
    class="preview-card"
    :class="{ 'preview-card--collapsed': isCollapsed }"
  >
    <PreviewCardHeader
      class="preview-card__header"
      :event-url="eventUrl"
      :event-type="event.type"
      :event-id="event.id"
      :tags="normalizedTags"
      :is-open="!isCollapsed"
      :is-visible-controls="isVisibleControls"
      @toggle-view="toggle"
      @delete="deleteEvent"
      @copy="copyCode"
      @download="downloadImage"
    />

    <div v-if="!isCollapsed" ref="event_body" class="preview-card__body">
      <slot />
    </div>

    <PreviewCardFooter
      v-if="!isCollapsed && (normalizedOrigin || event.serverName)"
      class="preview-card__footer"
      :server-name="event.serverName"
      :origin-config="normalizedOrigin"
    />
  </div>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { toPng, toBlob } from "html-to-image";
import download from "downloadjs";
import PreviewCardFooter from "~/components/PreviewCardFooter/PreviewCardFooter.vue";
import PreviewCardHeader from "~/components/PreviewCardHeader/PreviewCardHeader.vue";
import { NormalizedEvent } from "~/config/types";
import moment from "moment";
import { useNuxtApp } from "#app";
import { REST_API_URL } from "~/utils/events-transport";

export default defineComponent({
  components: {
    PreviewCardFooter,
    PreviewCardHeader,
  },
  props: {
    event: {
      type: Object as PropType<NormalizedEvent<unknown>>,
      required: true,
    },
  },
  setup() {
    if (process.client) {
      const { $events } = useNuxtApp();

      return {
        deleteEventById: $events?.removeById,
      };
    }

    return {
      deleteEventById: () => {},
    };
  },
  data() {
    return {
      isCollapsed: false,
      isVisibleControls: true,
    };
  },
  computed: {
    normalizedTags(): string[] {
      return [moment(this.event.date).format("HH:mm:ss"), ...this.event.labels];
    },
    normalizedOrigin(): unknown | null {
      const originEntriesList = Object.entries(this.event.origin || {})
        .map(([key, value]) => [key, String(value)])
        .filter(([_, value]) => Boolean(value));

      return originEntriesList.length > 0
        ? Object.fromEntries(originEntriesList)
        : null;
    },
    eventUrl(): string {
      return `${REST_API_URL}/api/event/${this.event.id}`;
    },
    fileName(): string {
      return `${this.event.type}-${this.event.id}.png`;
    },
  },
  methods: {
    toggle(): void {
      this.isCollapsed = !this.isCollapsed;
    },
    deleteEvent() {
      this.deleteEventById(this.event.id);
    },
    changeVisibleControls(value = true): void {
      this.isVisibleControls = value;
    },
    downloadImage() {
      this.changeVisibleControls(false);
      toPng(this.$refs.event as HTMLInputElement)
        .then((dataUrl) => {
          download(dataUrl, this.fileName);
        })
        .finally(() => {
          this.changeVisibleControls(true);
        });
    },
    copyCode(): void {
      this.changeVisibleControls(false);

      if (this.$refs.event) {
        toBlob(this.$refs.event as HTMLElement)
          .then((blob) => {
            if (blob) {
              navigator.clipboard.write([
                new ClipboardItem({ [blob.type]: blob }),
              ]);
            }
          })
          .catch((e) => console.error(e))
          .finally(() => {
            this.changeVisibleControls(true);
          });
      }
    },
  },
});
</script>

<style lang="scss" scoped>
.preview-card {
  @apply flex-grow flex flex-col p-3 lg:p-5 transition-colors dark:bg-gray-700;

  &:hover {
    @apply md:bg-gray-50 dark:bg-gray-900;
  }
}

.preview-card--collapsed {
  @apply shadow-bottom;
}

.preview-card__header {
  @apply w-full flex justify-between;
}

.preview-card__body {
  @apply flex flex-col mt-3;
}

.preview-card__footer {
  @apply w-full flex flex-col sm:flex-row sm:justify-between sm:items-center mt-3 text-xs text-gray-400;
}
</style>
