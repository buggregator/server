import { Meta, Story } from "@storybook/vue3";
import { normalizeFallbackEvent } from "~/utils/normalize-event";
import EventFallback from '~/components/EventFallback/EventFallback.vue';
import monologEventMock from '~/mocks/monolog.json'

export default {
  title: "Event/EventFallback",
  component: EventFallback
} as Meta<typeof EventFallback>;

const Template: Story = (args) => ({
  components: { EventFallback },
  setup() {
    return {
      args,
    };
  },
  template: `<event-fallback v-bind="args" />`,
});

export const Default = Template.bind({});

Default.args = {
  event: normalizeFallbackEvent({ ...monologEventMock, type: 'unknown' }),
};
