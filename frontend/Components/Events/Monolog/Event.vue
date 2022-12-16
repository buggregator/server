<template>
  <Event :event="event" class="event--monolog">
    <div class="event-monolog__wrap">
      <CodeSnippet class="event-monolog__snippet text-white mt-0" :code="event.text"/>
      <CodeSnippet v-if="hasPayloads" language="json" class="event-monolog__payloads" :code="event.payloads"/>
      <CodeSnippet v-if="hasFields" :title="field.title" v-for="field in fields" :key="field.title"
                   :code="field.value"/>
    </div>
  </Event>
</template>

<script>
import CodeSnippet from "@/Components/UI/CodeSnippet"
import Table from "@/Components/UI/Table"
import TableRow from "@/Components/UI/TableRow"
import Event from "../Event"

export default {
  components: {Event, TableRow, Table, CodeSnippet},
  props: {
    event: Object,
  },
  computed: {
    fields() {
      return this.event.fields
    },
    hasPayloads() {
      return this.event.payloads.constructor === Object &&
      Object.keys(this.event.payloads).length > 0
    },
    hasFields() {
      return this.fields.length > 0
    }
  },
}
</script>
