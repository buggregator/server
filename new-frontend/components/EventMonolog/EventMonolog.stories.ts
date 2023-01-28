import { normalizeMonologEvent } from "~/utils/normalizeEvent";
import { Meta, Story } from "@storybook/vue3";
import EventMonolog from './EventMonolog.vue';
import monologEventMock from '../../mocks/monolog.json'

export default {
  title: "Event/EventMonolog",
  component: EventMonolog,
  argTypes: {
    eventType: {
      control: { type: 'object' },
    },
  }
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

export const Default = Template.bind({});

Default.args = {
  event: normalizeMonologEvent(monologEventMock),
};
