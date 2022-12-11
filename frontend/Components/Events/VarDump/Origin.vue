<template>
  <div class="event-var-dump__footer">
    <span class="event-var-dump__footer-text" :title="`${origin.file}:${origin.line_number}`">
        {{ origin.name }}:{{ origin.line_number }}
    </span>
  </div>
</template>

<script>
import Label from "@/Components/UI/Label"

export default {
  components: {Label},
  props: {
    origin: Object,
  }
}
</script>
