<template>
  <section class="py-5 px-4 md:px-6 lg:px-8 border-b">
    <h3 class="text-muted font-bold uppercase text-sm mb-5">tags</h3>

    <div class="flex items-stretch flex-col md:flex-row mb-5">
      <div v-if="event.runtime.name"
           class="border border-purple-300 dark:border-gray-400 rounded px-4 pb-2 pt-1 hover:bg-purple-50 dark:hover:bg-purple-600 cursor-pointer md:mr-5 mb-3 md:mb-0">
        <span class="text-muted text-xs font-bold">runtime</span>
        <h4 class="font-bold">{{ event.runtime.name }}</h4>
        <p class="text-sm">Version: {{ event.runtime.version }}</p>
      </div>

      <div v-if="event.os.name"
           class="border border-purple-300 dark:border-gray-400 rounded px-4 pb-2 pt-1 hover:bg-purple-50 dark:hover:bg-purple-600 cursor-pointer md:mr-5 mb-3 md:mb-0">
        <span class="text-muted text-xs font-bold">os</span>
        <h4 class="font-bold">{{ event.os.name }}</h4>
        <p class="text-sm">Version: {{ event.os.version }}</p>
      </div>

      <div v-if="event.sdk.name"
           class="border border-purple-300 dark:border-gray-400 rounded px-4 pb-2 pt-1 hover:bg-purple-50 dark:hover:bg-purple-600 cursor-pointer md:mr-5 mb-3 md:mb-0">
        <span class="text-muted text-xs font-bold">sdk</span>
        <h4 class="font-bold">{{ event.sdk.name }}</h4>
        <p class="text-sm">Version: {{ event.sdk.version }}</p>
      </div>
    </div>

    <div class="flex flex-row flex-wrap items-center text-purple-600 dark:text-purple-100">
      <div class="flex border border-purple-300 rounded text-xs items-center mr-3 mb-2">
        <div class="px-3 py-1 border-r">env</div>
        <div class="px-3 py-1 bg-purple-100 dark:bg-purple-800 rounded-r font-bold">{{ event.environment }}</div>
      </div>
      <div class="flex border border-purple-300 rounded text-xs items-center mr-3 mb-2" v-if="event.logger">
        <div class="px-3 py-1 border-r">logger</div>
        <div class="px-3 py-1 bg-purple-100 dark:bg-purple-800 font-bold rounded-r">{{ event.logger }}</div>
      </div>
      <div class="flex border border-purple-300 rounded text-xs items-center mr-3 mb-2" v-if="event.os.name">
        <div class="px-3 py-1 border-r">os</div>
        <div class="px-3 py-1 bg-purple-100 dark:bg-purple-800 font-bold rounded-r">{{ event.os.name }}
          {{ event.os.version }}
        </div>
      </div>
      <div class="flex border border-purple-300 rounded text-xs items-center mr-3 mb-2" v-if="event.runtime.name">
        <div class="px-3 py-1 border-r">runtime</div>
        <div class="px-3 py-1 bg-purple-100 dark:bg-purple-800 font-bold rounded-r">{{ event.runtime.name }}
          {{ event.runtime.version }}
        </div>
      </div>
      <div class="flex border border-purple-300 rounded text-xs items-center mr-3 mb-2" v-if="event.serverName">
        <div class="px-3 py-1 border-r">server name</div>
        <div class="px-3 py-1 bg-purple-100 dark:bg-purple-800 font-bold rounded-r">{{ event.serverName }}</div>
      </div>
    </div>

    <CodeSnippet v-if="event.tags" class="mt-3" language="json" :code="event.tags"/>
  </section>
</template>

<script>
import CodeSnippet from "@/Components/UI/CodeSnippet"

export default {
  components: {CodeSnippet},
  props: {
    event: Object
  }
}
</script>
