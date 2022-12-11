<template>
  <div class="code-snippet">
    <pre :language="language" v-html="highlightCode"/>
    <button type="button" @click="doCopy" class="code-snippet__btn-copy" :class="{'active': copied}">
      <CopyIcon class="w-2 h-2 "/>copy
    </button>
  </div>
</template>

<script>
import {copyText} from 'vue3-clipboard'
import CopyIcon from "./Icons/CopyIcon";

const hljs = require('highlight.js')

export default {
  components: {CopyIcon},
  props: {
    code: String,
    language: {
      type: String,
      default: () => null
    }
  },
  data() {
    return {
      copied: false
    }
  },
  computed: {
    highlightCode() {
      return hljs.highlight(this.code, {language: this.language}).value
    }
  },
  methods: {
    doCopy() {
      this.copied = true
      setTimeout(() => this.copied = false, 100)

      let text = '';
      this.$slots.default().forEach(vnode => {
        text += vnode.children
      })

      copyText(text, undefined, (error, event) => {

      })
    }
  }
}
</script>
