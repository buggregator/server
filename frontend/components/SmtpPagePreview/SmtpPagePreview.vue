<template>
  <div class="smtp-page-preview">
    <div class="smtp-page-preview__in">
      <button
        class="smtp-page-preview__btn"
        :class="{
          'smtp-page-preview__btn--active': currentDevice === 'mobile',
        }"
        @click="currentDevice = 'mobile'"
      >
        <IconSvg class="smtp-page-preview__btn-icon" name="mobile-device" />
      </button>
      <button
        class="smtp-page-preview__btn"
        :class="{
          'smtp-page-preview__btn--active': currentDevice === 'tablet',
        }"
        @click="currentDevice = 'tablet'"
      >
        <IconSvg class="smtp-page-preview__btn-icon" name="tablet-device" />
      </button>
      <button
        class="smtp-page-preview__btn"
        :class="{
          'smtp-page-preview__btn--active': currentDevice === 'desktop',
        }"
        @click="currentDevice = 'desktop'"
      >
        <IconSvg class="smtp-page-preview__btn-icon" name="desktop-device" />
      </button>
    </div>
    <div class="smtp-page-preview__device" :class="deviceClassMod">
      <div><slot></slot></div>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import IconSvg from "~/components/IconSvg/IconSvg.vue";

export default defineComponent({
  components: {
    IconSvg,
  },
  props: {
    device: {
      type: String,
      default() {
        return "desktop";
      },
    },
  },
  data() {
    return {
      currentDevice: "desktop",
    };
  },
  computed: {
    deviceClassMod(): string {
      const deviceTypeClassMap: Record<string, string> = {
        desktop: "smtp-page-preview__device--desktop",
        tablet: "smtp-page-preview__device--tablet",
        mobile: "smtp-page-preview__device--mobile",
      };
      return deviceTypeClassMap[this.currentDevice];
    },
  },
  created() {
    this.currentDevice = this.device;
  },
});
</script>

<style lang="scss">
@import "assets/mixins";

.smtp-page-preview {
  @apply flex-1 flex flex-col items-center h-full;
}

.smtp-page-preview__in {
  @apply flex justify-center mb-5;
}

.smtp-page-preview__btn {
  @apply p-1 rounded;
}

.smtp-page-preview__btn--active {
  @apply bg-blue-50 text-blue-600;
}

.smtp-page-preview__btn-icon {
  @apply w-10 fill-current;
}

.smtp-page-preview__device {
  @apply flex-1 flex flex-col items-center bg-gray-50 dark:bg-gray-900;

  iframe {
    transition-property: width !important;
    @apply flex-1 rounded-md w-full h-full;
  }
}

.smtp-page-preview__device--desktop {
  @apply w-full h-full border rounded-md;

  > div {
    @apply flex-1 flex flex-col w-full h-full rounded-md bg-white ease-in duration-150;

    transition-property: width !important;

    > div {
      @apply flex-1 flex flex-col w-full h-full;
    }
  }
}

.smtp-page-preview__device--tablet {
  transition-property: width !important;
  width: auto;
  @apply justify-center border-2 rounded-3xl px-5 ease-in duration-150;

  &::after {
    @include border-style;
    @apply border bg-gray-100 dark:bg-gray-900 rounded-full block w-12 h-12 my-4;
    content: "";
  }

  &::before {
    @include border-style;
    @apply border bg-gray-100 dark:bg-gray-900 rounded-full block w-3 h-3 my-2;
    content: "";
  }

  > div {
    height: 1004px;
    width: 768px;
    transition-property: width !important;
    @apply border rounded-md bg-white ease-in duration-150;

    > div {
      @apply w-full h-full;
    }
  }
}

.smtp-page-preview__device--mobile {
  @apply justify-center border-2 rounded-3xl px-3;

  &::before {
    @include border-style;
    @apply border bg-gray-100 dark:bg-gray-900 rounded-full block w-3 h-3 my-2;
    content: "";
  }

  &::after {
    @include border-style;
    @apply border bg-gray-100 dark:bg-gray-900 rounded-full block w-8 h-8 my-3;
    content: "";
  }

  > div {
    height: 568px;
    width: 320px;
    transition-property: width !important;
    @apply border rounded-md bg-white ease-in duration-150;
    > div {
      @apply w-full h-full;
    }
  }
}
</style>
