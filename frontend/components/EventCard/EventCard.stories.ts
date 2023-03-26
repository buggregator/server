import { Meta, Story } from "@storybook/vue3";
import { EVENT_TYPES } from "~/config/constants";
import EventCard from "~/components/EventCard/EventCard.vue";

export default {
  title: "Event/EventCard",
  component: EventCard
}as Meta<typeof EventCard>;

const Template: Story = (args) => ({
  components: { EventCard },
  setup() {
    return {
      args,
    };
  },
  template: '<event-card v-bind="args">Hello world!</event-card>',
});

export const Default = Template.bind({});
Default.args = {
  event: {
    id: 'da076402-6f98-4ada-bae2-d77d405cf427',
    type: EVENT_TYPES.MONOLOG,
    serverName: "My server",
    origin: {
      one: 1,
      two: 2,
    },
    date: new Date(1673266869 * 1000),
    labels: ['Monolog', '200' ]
  },
};
