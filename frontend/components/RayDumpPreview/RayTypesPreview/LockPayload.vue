<template>
  <div class="ray-type-lock">
    <button :disabled="disabled" @click="continueExecution"
            class="btn btn--continue active:bg-grey-300">
            <span class="w-3 h-3 block">
                <svg xmlns="http://www.w3.org/2000/svg" width="100%" height="100%" viewBox="0 0 20 20">
                    <path fill="green" fill-rule="evenodd"
                          d="M16.75 10.83L4.55 19A1 1 0 0 1 3 18.13V1.87A1 1 0 0 1 4.55 1l12.2 8.13a1 1 0 0 1 0 1.7z"/>
                </svg>
            </span>

      <span>Continue</span>
    </button>
    <button :disabled="disabled" @click="stopExecution"
            class="btn btn--stop active:bg-grey-300">
      <span class="w-3 h-3 bg-red-700 block"></span>
      <span>Stop execution</span>
    </button>
  </div>
</template>

<script lang="ts">
import {defineComponent, PropType} from "vue";
import {RayPayload} from "~/config/types";
import {apiTransport} from '~/utils/events-transport'

const {
  rayStopExecution,
  rayContinueExecution
} = apiTransport({onEventReceiveCb: () => {}})

export default defineComponent({
  props: {
    payload: {
      type: Object as PropType<RayPayload>,
      required: true,
    },
  },
  data() {
    return {
      disabled: false,
    }
  },
  methods: {
    continueExecution() {
      this.disabled = true
      rayContinueExecution(this.hash)
    },
    stopExecution() {
      this.disabled = true
      rayStopExecution(this.hash)
    }
  },
  computed: {
    hash() {
      return this.payload.content.name
    }
  }
});
</script>

<style lang="scss" scoped>
.ray-type-lock {
  @apply flex items-center mt-3;
}

.btn {
  @apply px-5 py-2 flex items-center space-x-3 bg-gray-100 dark:bg-gray-800 border dark:border-gray-600 text-sm font-medium hover:bg-gray-50 focus:outline-none disabled:opacity-50;

  &--continue {
    @apply rounded-l-full;
  }

  &--stop {
    @apply border-l-0 rounded-r-full;
  }
}
</style>
