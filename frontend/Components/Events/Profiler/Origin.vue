<template>
  <div class="event-profiler__footer">
    <span>
        <Label>application: {{ origin.app }}</Label>
        <Label v-for="(value, tag) in origin.tags" v-if="value">{{ tag }}: {{ value }}</Label>
    </span>

    <Host :name="origin.hostname" class="event-profiler__host"/>
  </div>
</template>

<script>
import Label from "@/Components/UI/Label"
import Host from "@/Components/UI/Host"

export default {
  components: {Label, Host},
  props: {
    origin: Object,
  }
}
</script>
