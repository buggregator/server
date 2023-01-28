<template>
  <div class="code-snippet">
    <highlightjs :language="language" :code="normalizedCode" />
    <button
      type="button"
      class="code-snippet__copy"
      :class="{ 'code-snippet__copy--active': isCopied }"
      @click="copyCode"
    >
      <IconSvg name="copy" class="code-snippet__copy-icon" />
      copy
    </button>
  </div>
</template>

<script lang="ts">
import IconSvg from "~/components/IconSvg/IconSvg.vue";
import hljs from "highlight.js/lib/common";
import hljsVuePlugin from "@highlightjs/vue-plugin";

export default {
  components: {
    IconSvg,
    highlightjs: hljsVuePlugin.component,
  },
  props: {
    code: {
      type: [String, Object],
      required: true,
    },
    language: {
      type: String,
      default: "plaintext",
    },
  },
  data() {
    this.hljs = hljs;

    return {
      isCopied: false,
    };
  },
  computed: {
    normalizedCode(): String {
      return typeof this.code === "string"
        ? this.code
        : JSON.stringify(this.code, null, " ");
    },
  },
  methods: {
    copyCode(): Promise<void> {
      this.isCopied = true;

      navigator.clipboard
        .writeText(this.code)
        .then(() => {
          setTimeout(() => {
            this.isCopied = false;
          }, 200);
        })
        .catch((e) => {
          console.error(e);
        });
    },
  },
};
</script>

<style lang="scss" scoped>
.code-snippet {
  @apply relative bg-gray-200 dark:bg-gray-800;
}

.code-snippet__copy {
  @apply flex rounded-full items-center gap-x-1 absolute top-2 right-2 px-2 bg-white dark:bg-gray-900 border text-gray-600 transition-all text-xs font-bold border-gray-600;

  &:hover {
    @apply border-gray-200 text-white bg-gray-900 dark:bg-white;
  }
}

.code-snippet__copy--active {
  @apply transform scale-110 bg-green-500 hover:bg-green-500 transition-colors;
}

.code-snippet__copy-icon {
  @apply w-2 h-2;
}
</style>
