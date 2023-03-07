<template>
  <div class="event-header">
    <div class="event-header__tags">
      <a
        v-if="eventUrl"
        :href="eventUrl"
        target="_blank"
        class="event-header__tags-chip"
      >
        JSON
      </a>

      <template v-if="tags.length">
        <div
          v-for="tag in tags"
          :key="tag"
          ref="tags"
          class="event-header__tag"
          :class="`event-header__tag--${eventType}`"
        >
          {{ tag }}
        </div>
      </template>
    </div>

    <div v-if="isVisibleControls" class="event-header__container">
      <button
        class="event-header__button event-header__button--copy"
        @click="onCopyButtonClick"
        @click.right.prevent="onCopyButtonRightClick"
      >
        <icon-svg name="copy" class="event-header__button-icon" />
      </button>

      <button
        class="event-header__button event-header__button--collapse"
        :class="`event-header__button--${eventType}`"
        @click="changeView"
      >
        <icon-svg
          v-if="isOpen"
          name="minus"
          class="event-header__button-icon"
        />
        <icon-svg
          v-if="!isOpen"
          name="plus"
          class="event-header__button-icon"
        />
      </button>

      <button
        class="event-header__button event-header__button--delete"
        @click="onDeleteButtonClick"
      >
        <icon-svg name="times" class="event-header__button-icon" />
      </button>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { EVENT_TYPES } from "~/config/constants";
import { OneOfValues } from "~/config/types";
import IconSvg from "~/components/IconSvg/IconSvg.vue";

export default defineComponent({
  components: {
    IconSvg,
  },
  props: {
    eventUrl: {
      type: String,
      default: "",
    },
    eventType: {
      type: String,
      validator: (val: OneOfValues<typeof EVENT_TYPES>) =>
        Object.values(EVENT_TYPES).includes(val),
      required: true,
    },
    eventId: {
      type: String,
      required: true,
    },
    tags: {
      type: Array as PropType<string[]>,
      default: () => [],
    },
    isOpen: {
      type: Boolean,
      default: true,
    },
    isVisibleControls: {
      type: Boolean,
      default: true,
    },
  },
  emits: {
    delete(payload: boolean) {
      return payload;
    },
    toggleView(payload: boolean) {
      return payload;
    },
    copy(payload: boolean) {
      return payload;
    },
    download(payload: boolean) {
      return payload;
    },
  },
  data() {
    return {
      exportableEl: null,
    };
  },
  methods: {
    calcModByEventType(defaultClassName: string): string {
      return `${defaultClassName}--${this.eventType}`;
    },
    changeView() {
      this.$emit("toggleView", true);
    },
    onDeleteButtonClick() {
      this.$emit("delete", true);
    },
    onCopyButtonRightClick() {
      this.$emit("copy", true);
    },
    onCopyButtonClick() {
      this.$emit("download", true);
    },
  },
});
</script>

<style lang="scss" scoped>
$eventTypeColorsMap: (
  "var-dump" "red",
  "smtp" "orange",
  "sentry" "pink",
  "profiler" "purple",
  "monolog" "gray",
  "inspector" "gray",
  "ray" "gray",
  "unknown" "gray"
);

.event-header {
  @apply w-full flex justify-between;
}

.event-header__container {
  @apply flex items-center space-x-2;
}

.event-header__tags {
  @apply flex flex-wrap gap-3;
}

.event-header__tags-chip {
  @apply font-bold px-2 rounded-full text-xs inline-flex items-center transition-colors text-blue-700 bg-blue-100 border hover:bg-blue-700 hover:text-white;
}

.event-header__tag {
  @apply font-bold px-2 rounded-full text-xs inline-flex items-center transition-colors border cursor-help;

  /* Applied tailwind classes depends on event type
   Need to keep declaration for tailwind correct work:
   'var-dump' 'bg-red-50 dark:bg-red-700 text-red-800 dark:text-red-50 dark:border-red-600' 'bg-red-100 dark:bg-red-500',
   'Smtp' 'bg-orange-50 dark:bg-orange-700 text-orange-800 dark:text-orange-50 dark:border-orange-600' 'bg-orange-100 dark:bg-orange-500',
   'Sentry' 'bg-pink-50 dark:bg-pink-700 text-pink-800 dark:text-pink-50 dark:border-pink-600' 'bg-pink-100 dark:bg-pink-500',
   'profiler' 'bg-purple-50 dark:bg-purple-700 text-purple-800 dark:text-purple-50 dark:border-purple-600' 'bg-purple-100 dark:bg-purple-500',
   'monolog' 'bg-gray-50 dark:bg-gray-700 text-gray-800 dark:text-gray-50 dark:border-gray-600' 'bg-gray-100 dark:bg-gray-500',
   'inspector' 'bg-gray-50 dark:bg-gray-700 text-gray-800 dark:text-gray-50 dark:border-gray-600' 'bg-gray-100 dark:bg-gray-500',
   'ray' 'bg-gray-50 dark:bg-gray-700 text-gray-800 dark:text-gray-50 dark:border-gray-600' 'bg-gray-100 dark:bg-gray-500' */

  @each $map in $eventTypeColorsMap {
    $name: nth($map, 1);
    $color: nth($map, 2);

    &--#{$name} {
      @apply bg-#{$color}-50 dark:bg-#{$color}-700 text-#{$color}-800 dark:text-#{$color}-50 dark:border-#{$color}-600;

      &:hover {
        @apply bg-#{$color}-100 dark:bg-#{$color}-500;
      }
    }
  }
}

.event-header__button {
  @apply w-5 h-5 md:w-4 md:h-4 rounded-full opacity-90 hover:opacity-100 transition transition-all hover:ring-4 ring-offset-1;
  /* Applied tailwind classes depends on event type
   Need to keep declaration for tailwind correct work:
   'var-dump' 'bg-red-600 ring-red-300',
   'Smtp' 'bg-orange-600 ring-orange-300',
   'Sentry' 'bg-pink-600 ring-pink-300',
   'profiler' 'bg-purple-600 ring-purple-300',
   'monolog' 'bg-gray-600 ring-gray-300',
   'inspector' 'bg-gray-600 ring-gray-300',
   'ray' 'bg-gray-600 ring-gray-300' */

  @each $map in $eventTypeColorsMap {
    $name: nth($map, 1);
    $color: nth($map, 2);

    &--#{$name} {
      @apply bg-#{$color}-600 ring-#{$color}-300;
    }
  }
}

.event-header__button--collapse {
  @apply text-white bg-gray-600 ring-gray-300;
}

.event-header__button--delete {
  @apply text-red-700 bg-white hover:bg-red-700 hover:text-white;
}

.event-header__button-icon {
  @apply p-1;
}
</style>
