import { Meta, Story } from "@storybook/vue3";
import { normalizeMonologEvent } from "~/utils/normalize-event";
import EventMonolog from '~/components/EventMonolog/EventMonolog.vue';
import monologEventMock from '~/mocks/monolog.json'

export default {
  title: "Event/EventMonolog",
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

export const Default = Template.bind({});

Default.args = {
  event: normalizeMonologEvent(monologEventMock),
};
