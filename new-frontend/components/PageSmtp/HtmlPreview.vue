<template>
  <div class="html-preview">
    <div class="html-preview__wrap">
      <button class="html-preview__btn" :class="{'active': currentDevice == 'mobile'}" @click="currentDevice = 'mobile'">
        <IconSvg class="html-preview__btn-icon" name="mobile-device"/>
      </button>
      <button class="html-preview__btn" :class="{'active': currentDevice == 'tablet'}" @click="currentDevice = 'tablet'">
        <IconSvg class="html-preview__btn-icon" name="tablet-device"/>
      </button>
      <button class="html-preview__btn" :class="{'active': currentDevice == 'desktop'}" @click="currentDevice = 'desktop'">
        <IconSvg class="html-preview__btn-icon" name="desktop-device"/>
      </button>
    </div>
    <div :class="`device-${currentDevice}`">
      <div>
        <slot></slot>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import {defineComponent, PropType} from "vue";
import IconSvg from "~/components/IconSvg/IconSvg.vue";

export default defineComponent({
  components: {
    IconSvg
  },
  props: {
    device: {
      type: String,
      default() {
        return 'desktop'
      }
    }
  },
  data() {
    return {
      currentDevice: 'desktop'
    }
  },
  created() {
    this.currentDevice = this.device
  }
})
</script>

<style lang="scss">
@import "assets/mixins";

.html-preview {
  @apply flex flex-col items-center h-full;

  &__wrap {
    @apply flex justify-center mb-5;
  }

  &__btn {
    @apply p-1 rounded;

    &.active {
      @apply bg-blue-50 text-blue-600;
    }
  }

  &__btn-icon {
    @apply w-10 fill-current;
  }
}

.device-desktop {
  @apply w-full h-full border rounded-md;
}

.device-desktop > div {
  @apply w-full h-full rounded-md bg-white;
}

.device-tablet {
  transition-property: width!important;
  width: auto;
  @apply justify-center border-2 rounded-3xl px-5 ease-in duration-150;
}

.device-tablet::after {
  @include border-style;
  @apply border bg-gray-100 dark:bg-gray-900 rounded-full block w-12 h-12 my-4;
  content: "";
}

.device-tablet::before {
  @include border-style;
  @apply border bg-gray-100 dark:bg-gray-900 rounded-full block w-3 h-3 my-2;
  content: "";
}

.device-tablet > div {
  height: 1004px;
  width: 768px;
}

.device-mobile {
  @apply justify-center border-2 rounded-3xl px-3;
}

.device-mobile::before {
  @include border-style;
  @apply border bg-gray-100 dark:bg-gray-900 rounded-full block w-3 h-3 my-2;
  content: "";
}

.device-mobile::after {
  @include border-style;
  @apply border bg-gray-100 dark:bg-gray-900 rounded-full block w-8 h-8 my-3;
  content: "";
}

.device-mobile > div {
  height: 568px;
  width: 320px;
}

.device-desktop,
.device-tablet,
.device-mobile {
  @apply flex flex-col items-center bg-gray-50 dark:bg-gray-900;
}

.device-desktop > div {
  transition-property: width !important;
  @apply ease-in duration-150;
}

.device-tablet > div,
.device-mobile > div {
  transition-property: width !important;
  @apply border rounded-md bg-white ease-in duration-150;
}

.device-desktop iframe,
.device-tablet iframe,
.device-mobile iframe {
  transition-property: width !important;
  /*@apply ease-in duration-300;*/
  @apply rounded-md w-full h-full;
}
</style>
