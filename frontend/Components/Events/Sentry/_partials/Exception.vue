<template>
  <div class="sentry-exception">
    <NuxtLink tag="div" :to="event.route.show" class="sentry-exception__link flex-grow">
      <h3 class="sentry-exception__title">
        {{ event.payload.type }}
      </h3>
      <pre class="sentry-exception__text" v-html="event.payload.value"/>
    </NuxtLink>
    <div class="sentry-exception__files w-full" v-if="hasFrames">
      <File :key="`${file.filename}-${file.lineno}-${i}`"
            :file="file"
            v-for="(file, i) in stacktrace"
            :collapsed="i !== 0" class="sentry-exception__file"
      />
    </div>
  </div>
</template>

<script>
import File from "../_partials/File"

export default {
  components: {File},
  props: {
    event: Object,
    frames: {
      type: Number,
      default: () => 5
    }
  },
  computed: {
    hasFrames() {
      return this.frames > 0
    },
    stacktrace() {
      return this.event.stacktrace.slice(0, this.frames)
    }
  }
}
</script>
