<script>
import Event from "@/Components/Events/Inspector/Event"
import EventsPage from "@/mixins/EventsPage"

export default {
  components: {Event},
  extends: EventsPage,
  data() {
    return {
      title: 'Inspector events',
      type: 'inspector',
    }
  },
}
</script>
