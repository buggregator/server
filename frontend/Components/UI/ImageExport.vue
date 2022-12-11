<template>
  <span @click="download" class="download-chip">png</span>
</template>

<script>
import {copyText} from 'vue3-clipboard'
import {toPng, toBlob} from 'html-to-image'
import download from "@/Utils/download"

export default {
  props: {
    name: String,
    el: HTMLElement,
  },
  methods: {
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

      toBlob(this.el)
        .then(async function (blob) {
          await navigator.clipboard.write([
            new ClipboardItem({
              [blob.type]: blob
            })
          ])
        })
    }
  }
}
</script>
