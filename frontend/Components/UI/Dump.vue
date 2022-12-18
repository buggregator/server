<template>
  <CodeSnippet v-if="isBool" language="php" :code="value == 1"/>
  <div v-else v-html="cleanHtml" class="dump" :class="{'active': dumpId}"></div>
</template>

<style lang="scss">
.dump {
  @apply border-gray-300 dark:border-gray-500 divide-gray-300 dark:divide-gray-500 font-mono p-2 md:px-3 lg:px-4 border bg-gray-200 dark:bg-gray-800 text-blue-700 dark:text-white text-sm break-all text-2xs sm:text-xs md:text-sm lg:text-base;
}
</style>

<script>
import CodeSnippet from "@/Components/UI/CodeSnippet"

export default {
  components: {CodeSnippet},
  props: ['value', 'type'],
  data() {
    return {
      evaluated: false,
    }
  },
  mounted() {
    if (this.dumpId) {
      window.Sfdump(this.dumpId)
    }
  },
  computed: {
    isBool() {
      return typeof this.value == 'boolean' || this.type === 'boolean'
    }
    ,
    dumpId() {
      if (typeof this.value === 'string') {
        const matches = this.value.match(/(sf\-dump\-[0-9]+)/i)
        if (matches) {
          return matches[0]
        }
      }

      return null
    }
    ,
    cleanHtml() {
      if (this.dumpId) {
        // Remove all style and script tags from dump
        return this.value.replace(
          /<(style|script)\b[^<]*(?:(?!<\/style>)<[^<]*)*<\/(style|script)>/gi,
          ""
        )
      }

      return this.value
    }
  }
}
</script>

<style>
pre.sf-dump {
  display: block;
  white-space: pre;
  padding: 5px;
  overflow: initial !important;
}

pre.sf-dump:after {
  content: "";
  visibility: hidden;
  display: block;
  height: 0;
  clear: both;
}

pre.sf-dump span {
  display: inline;
}

pre.sf-dump a {
  text-decoration: none;
  cursor: pointer;
  border: 0;
  outline: none;
  color: inherit;
}

pre.sf-dump img {
  max-width: 50em;
  max-height: 50em;
  margin: .5em 0 0 0;
  padding: 0;
  background: url(data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAAAAAA6mKC9AAAAHUlEQVQY02O8zAABilCaiQEN0EeA8QuUcX9g3QEAAjcC5piyhyEAAAAASUVORK5CYII=) #D3D3D3;
}

pre.sf-dump .sf-dump-ellipsis {
  display: inline-block;
  text-overflow: ellipsis;
  max-width: 5em;
  white-space: nowrap;
  overflow: hidden;
  vertical-align: top;
}

pre.sf-dump .sf-dump-ellipsis + .sf-dump-ellipsis {
  max-width: none;
}

pre.sf-dump code {
  display: inline;
  padding: 0;
  background: none;
}

.sf-dump-public.sf-dump-highlight, .sf-dump-protected.sf-dump-highlight, .sf-dump-private.sf-dump-highlight, .sf-dump-str.sf-dump-highlight, .sf-dump-key.sf-dump-highlight {
  background: rgba(111, 172, 204, 0.3);
  border: 1px solid #7DA0B1;
  border-radius: 3px;
}

.sf-dump-public.sf-dump-highlight-active, .sf-dump-protected.sf-dump-highlight-active, .sf-dump-private.sf-dump-highlight-active, .sf-dump-str.sf-dump-highlight-active, .sf-dump-key.sf-dump-highlight-active {
  background: rgba(253, 175, 0, 0.4);
  border: 1px solid #ffa500;
  border-radius: 3px;
}

pre.sf-dump .sf-dump-search-hidden {
  display: none !important;
}

pre.sf-dump .sf-dump-search-wrapper {
  font-size: 0;
  white-space: nowrap;
  margin-bottom: 5px;
  display: flex;
  position: -webkit-sticky;
  position: sticky;
  top: 5px;
}

pre.sf-dump .sf-dump-search-wrapper > * {
  vertical-align: top;
  box-sizing: border-box;
  height: 21px;
  font-weight: normal;
  border-radius: 0;
  background: #FFF;
  color: #757575;
  border: 1px solid #BBB;
}

pre.sf-dump .sf-dump-search-wrapper > input.sf-dump-search-input {
  padding: 3px;
  height: 21px;
  font-size: 12px;
  border-right: none;
  border-top-left-radius: 3px;
  border-bottom-left-radius: 3px;
  color: #000;
  min-width: 15px;
  width: 100%;
}

pre.sf-dump .sf-dump-search-wrapper > .sf-dump-search-input-next, pre.sf-dump .sf-dump-search-wrapper > .sf-dump-search-input-previous {
  background: #F2F2F2;
  outline: none;
  border-left: none;
  font-size: 0;
  line-height: 0;
}

pre.sf-dump .sf-dump-search-wrapper > .sf-dump-search-input-next {
  border-top-right-radius: 3px;
  border-bottom-right-radius: 3px;
}

pre.sf-dump .sf-dump-search-wrapper > .sf-dump-search-input-next > svg, pre.sf-dump .sf-dump-search-wrapper > .sf-dump-search-input-previous > svg {
  pointer-events: none;
  width: 12px;
  height: 12px;
}

pre.sf-dump .sf-dump-search-wrapper > .sf-dump-search-count {
  display: inline-block;
  padding: 0 5px;
  margin: 0;
  border-left: none;
  line-height: 21px;
  font-size: 12px;
}

pre.sf-dump, pre.sf-dump .sf-dump-default {
  color: #FF8400;
  line-height: 1.2em;
  word-wrap: break-word;
  white-space: pre-wrap;
  position: relative;
  z-index: 10;
  word-break: break-all
}

pre.sf-dump .sf-dump-num {
  font-weight: bold;
  color: #1299DA
}

pre.sf-dump .sf-dump-const {
  font-weight: bold
}

pre.sf-dump .sf-dump-str {
  font-weight: bold;
  color: #56DB3A
}

pre.sf-dump .sf-dump-note {
  color: #1299DA
}

pre.sf-dump .sf-dump-ref {
  color: #A0A0A0
}

pre.sf-dump .sf-dump-public {
  color: #FFFFFF
}

pre.sf-dump .sf-dump-protected {
  color: #FFFFFF
}

pre.sf-dump .sf-dump-private {
  color: #FFFFFF
}

pre.sf-dump .sf-dump-meta {
  color: #B729D9
}

pre.sf-dump .sf-dump-key {
  color: #56DB3A
}

pre.sf-dump .sf-dump-index {
  color: #1299DA
}

pre.sf-dump .sf-dump-ellipsis {
  color: #FF8400
}

pre.sf-dump .sf-dump-ns {
  user-select: none;
}

pre.sf-dump .sf-dump-ellipsis-note {
  color: #1299DA
}
</style>
