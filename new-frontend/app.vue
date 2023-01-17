<template>
  <div style="padding: 20px">
    <event-header
      event-url="https://storybook.js.org/docs/vue/writing-stories/args"
      event-type="Sentry"
      :tags="['one', 'two', 'tree']"
      :is-open="true"
      event-id="test-id"
    />
    &lt;! -- body --&gt;
    <event-footer
      :origin-config="{ key1: 'value1', key2: 'value2' }"
      server-name="Best server"
    />
  </div>
</template>

<script lang="ts" setup>
import "./assets/tailwind.css";
</script>
