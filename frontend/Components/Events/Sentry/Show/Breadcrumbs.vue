<template>
  <section class="py-5 px-4 md:px-6 lg:px-8 border-b" v-if="hasBreadcrumbs">
    <h3 class="text-muted font-bold uppercase text-sm mb-5">breadcrumbs</h3>

    <div style="height: 400px;" class="flex flex-col border border-purple-300 dark:border-purple-700 rounded">
      <nav style="grid-template-columns: 1fr 100px 200px 17px"
           class="border-b border-purple-300 dark:border-purple-700 grid bg-purple-50 dark:bg-purple-800 text-xs font-bold text-purple-600 dark:text-purple-100 rounded-t">
        <div class="p-3 uppercase">description</div>
        <div class="p-3 uppercase">level</div>
        <div class="p-3 uppercase">time</div>
      </nav>
      <div
        class="bg-gray-100 dark:bg-gray-800 max-h-full flex-1 overflow-y-scroll divide-y divide-purple-300 dark:divide-purple-600">
        <div style="grid-template-columns: 1fr 100px 200px" class="grid text-xs" v-for="b in event.breadcrumbs">
          <div class="p-3">
            <p class="font-bold">{{ b.message }}</p>

            <div class="flex flex-row flex-wrap items-center text-purple-600 dark:text-purple-100 text-2xs my-3">
              <div class="flex border border-purple-300 dark:border-purple-700 rounded items-center mr-3 mb-2">
                <div class="px-2 border-r dark:border-purple-700">type</div>
                <div class="px-2 bg-purple-100 dark:bg-purple-800 rounded-r font-bold">{{ b.type }}</div>
              </div>
              <div class="flex border border-purple-300 dark:border-purple-700 rounded items-center mr-3 mb-2">
                <div class="px-2 border-r dark:border-purple-700">category</div>
                <div class="px-2 bg-purple-100 dark:bg-purple-800 rounded-r font-bold">{{ b.category }}</div>
              </div>
            </div>

            <CodeSnippet v-if="b.data" class="mt-3" language="json" :code="b.data"/>
          </div>
          <div class="p-3">
            <Label class="px-3 py-1">{{ b.level }}</Label>
          </div>
          <div class="p-3">{{ date(b.timestamp).fromNow() }}</div>
        </div>
      </div>
    </div>
  </section>
</template>

<script>
import Dump from "@/Components/UI/Dump";
import CodeSnippet from "@/Components/UI/CodeSnippet";
import Label from "@/Components/UI/Label";

export default {
  components: {CodeSnippet, Dump, Label},
  props: {
    event: Object
  },
  methods: {
    date(timestamp) {
      return this.$moment.unix(timestamp)
    }
  },
  computed: {
    hasBreadcrumbs() {
      return this.event.breadcrumbs.length > 0
    }
  }
}
</script>
