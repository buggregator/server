<template>
  <div class="smtp-page page">
    <div ref="header" class="breadcrumbs">
      <div class="breadcrumbs_item current">Mailbox</div>
    </div>
    <main>
      <div class="smtp-page_sidebar">
        <PerfectScrollbar :style="{height: menuHeight}" v-if="hasEvents">
          <NavItem v-for="event in events" :event="event" :key="event.uuid"/>
        </PerfectScrollbar>

        <div v-else class="smtp-page_sidebar-empty">
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 35 36">
            <path fill-rule="evenodd" clip-rule="evenodd" d="M26 4h-4V2h4v2Zm1 2h-5v2h5a8 8 0 0 1 8 8v11a3 3 0 0 1-3 3H20v5c0 .6-.4 1-1 1h-4a1 1 0 0 1-1-1v-5H3a3 3 0 0 1-3-3V16a8 8 0 0 1 8-8h12V1c0-.6.4-1 1-1h6c.6 0 1 .4 1 1v4c0 .6-.4 1-1 1Zm-7 8v-4h-6.7a8 8 0 0 1 2.7 6v12h16c.6 0 1-.4 1-1V16a6 6 0 0 0-6-6h-5v4a1 1 0 1 1-2 0Zm-4 16h2v4h-2v-4Zm-2-14v12H3a1 1 0 0 1-1-1V16a6 6 0 0 1 12 0Zm-9 3a1 1 0 1 0 0 2h5.5a1 1 0 1 0 0-2H5Z"/>
          </svg>
          <h3>Your inbox is empty</h3>
        </div>
      </div>

      <section>
        <h2>SMTP Settings</h2>
        <p class="smtp-page_description">Use these settings to send messages directly from your email client or mail transfer agent.</p>

        <div v-if="hasEvents" class="my-3">
          <button @click="clearEvents" class="events__btn-clear">Clear events</button>
        </div>
      </section>
    </main>
  </div>
</template>

<script>
import {PerfectScrollbar} from 'vue2-perfect-scrollbar'
import NavItem from "@/Components/Events/Smtp/NavItem"
import Info from "@/Components/Events/Smtp/Info"

export default {
  components: {
    NavItem, PerfectScrollbar, Info
  },
  data() {
    return {
      menuHeight: 0
    }
  },
  mounted() {
    this.$store.dispatch('events/fetch')
    this.calculateMenuHeight()
    window.addEventListener('resize', (event) => {
      this.calculateMenuHeight()
    });
  },
  computed: {
    events() {
      return this.$store.getters['events/filteredByType']('smtp')
    },
    hasEvents() {
      return this.events.length > 0
    }
  },
  methods: {
    clearEvents() {
      this.store.dispatch('events/clear', 'sentry')
    },
    calculateMenuHeight() {
      const headerHeight = this.$refs.header ? parseInt(this.$refs.header.offsetHeight) : 0
      this.menuHeight = (window.innerHeight - headerHeight) + 'px'
    }
  }
}
</script>
