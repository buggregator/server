<template>
  <div
    :id="event.id"
    ref="event"
    class="event"
    :class="{ 'event--collapsed': isCollapsed }"
  >
    <event-header
      class="event__header"
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

    <div v-if="!isCollapsed" ref="event_body" class="event__body">
      <slot />
    </div>

    <event-footer
      v-if="!isCollapsed && (normalizedOrigin || event.serverName)"
      class="event__footer"
      :server-name="event.serverName"
      :origin-config="normalizedOrigin"
    />
  </div>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { toPng, toBlob } from "html-to-image";
import download from "downloadjs";
import EventFooter from "~/components/EventFooter/EventFooter.vue";
import EventHeader from "~/components/EventHeader/EventHeader.vue";
import { NormalizedEvent } from "~/config/types";
import moment from "moment";

export default defineComponent({
  components: {
    EventFooter,
    EventHeader,
  },
  props: {
    event: {
      type: Object as PropType<NormalizedEvent>,
      required: true,
    },
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
    normalizedOrigin(): Object | null {
      return Object.keys(this.event.origin || {}).length > 0
        ? this.event.origin
        : null;
    },
    eventUrl(): string {
      return `/api/event/${this.event.id}`;
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
      // this.$store.dispatch("events/delete", this.event);
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
.event {
  @apply flex-grow flex flex-col p-3 lg:p-5 transition-colors md:dark:bg-gray-700;

  &:hover {
    @apply md:bg-gray-50 dark:bg-gray-900;
  }
}

.event--collapsed {
  @apply shadow-bottom;
}

.event__header {
  @apply w-full flex justify-between;
}

.event__body {
  @apply flex flex-col mt-3;
}

.event__footer {
  @apply w-full flex flex-col sm:flex-row sm:justify-between sm:items-center mt-3 text-xs text-gray-400;
}
</style>
