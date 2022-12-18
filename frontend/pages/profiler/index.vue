<script>
import Event from "@/Components/Events/Profiler/Event"
import EventsPage from "@/mixins/EventsPage"

export default {
  components: {Event},
  extends: EventsPage,
  data() {
    return {
      title: 'Profiler events',
      type: 'profiler',
    }
  },
}
</script>
