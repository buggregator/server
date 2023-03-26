import { Meta, Story } from "@storybook/vue3";
import { normalizeMonologEvent } from "~/utils/normalize-event";
import EventMonolog from '~/components/EventMonolog/EventMonolog.vue';
import monologEventMock from '~/mocks/monolog.json'

export default {
  title: "Monolog",
  component: EventMonolog
} as Meta<typeof EventMonolog>;

const Template: Story = (args) => ({
  components: { EventMonolog },
  setup() {
    return {
      args,
    };
  },
  template: `<event-monolog v-bind="args" />`,
});

export const Event = Template.bind({});

Event.args = {
  event: normalizeMonologEvent(monologEventMock),
};
