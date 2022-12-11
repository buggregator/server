<template>
  <div class="sentry-file" @click="collapsed = !collapsed">
    <div class="sentry-file__title-wrap">
      <div class="sentry-file__title">
        {{ file.filename }}
        <span v-if="file.function" class="sentry-exception__title-info">in</span>
        {{ file.function || null }}
        <span v-if="file.function" class="sentry-exception__title-info">at line</span>
        {{ file.lineno }}
      </div>
      <div class="sentry-file__title-icon" v-if="file.pre_context">
        <svg viewBox="0 0 16 16"
             fill="currentColor"
             height="100%" width="100%"
             :class="{'rotate': collapsed}"
        >
          <path
            d="M14,11.75a.74.74,0,0,1-.53-.22L8,6.06,2.53,11.53a.75.75,0,0,1-1.06-1.06l6-6a.75.75,0,0,1,1.06,0l6,6a.75.75,0,0,1,0,1.06A.74.74,0,0,1,14,11.75Z"></path>
        </svg>
      </div>
    </div>
    <div class="sentry-file__body" v-if="file.pre_context && !collapsed">
      <div class="sentry-file__row" v-for="(line, i) in file.pre_context">
        <div class="sentry-file__row-number">{{ file.lineno - (file.pre_context.length - i) }}.</div>
        <pre class="sentry-file__row-text" v-html="line"></pre>
      </div>

      <div class="sentry-file__row sentry-file__row--red">
        <div class="w-12">{{ file.lineno }}.</div>
        <pre v-html="file.context_line"></pre>
      </div>
      <div class="sentry-file__row" v-for="(line, i) in file.post_context">
        <div class="sentry-file__row-number">{{ file.lineno + i + 1 }}.</div>
        <pre class="sentry-file__row-text" v-html="line"></pre>
      </div>
      <div class="event-table" v-if="file.vars">
        <div class="event-table__row" v-for="(v, k) in file.vars">
          <div class="event-table__cell-name">{{ k }}</div>
          <div class="event-table__cell-value">{{ v }}</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  props: {
    file: Object,
    collapsed: {
      type: Boolean,
      default: true
    }
  }
}
</script>
