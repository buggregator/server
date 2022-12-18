<template>
  <button @click="copy" @click.right="download" class="button button__copy">
    <CopyIcon/>
  </button>
</template>

<script>
import CopyIcon from "./Icons/CopyIcon"
import {toPng, toBlob} from 'html-to-image'
import download from "@/Utils/download"

export default {
  components: {CopyIcon},
  props: {
    name: String,
    el: HTMLElement,
  },
  methods: {
    copy() {
      this.$store.commit('theme/takeScreenshot', true)
      toBlob(this.el)
        .then(async function (blob) {
          await navigator.clipboard.write([
            new ClipboardItem({
              [blob.type]: blob
            })
          ])
        })
        .finally(() => {
          this.$store.commit('theme/takeScreenshot', false)
        })
    },
    download() {
      this.$store.commit('theme/takeScreenshot', true)
      const name = this.name
      toPng(this.el)
        .then(function (dataUrl) {
          download(dataUrl, `${name}.png`)
        })
        .finally(() => {
          this.$store.commit('theme/takeScreenshot', false)
        })
    }
  }
}
</script>
