import { normalizeSentryEvent } from "~/utils/normalizeEvent";
import { Meta, Story } from "@storybook/vue3";
import EventSentryFrame from './EventSentryFrame.vue';
import sentryEventMock from '../../mocks/sentry.json'

export default {
  title: "Event/EventSentryFrame",
  component: EventSentryFrame
} as Meta<typeof EventSentryFrame>;

const Template: Story = (args) => ({
  components: { EventSentryFrame },
  setup() {
    return {
      args,
    };
  },
  template: `<event-sentry-frame v-bind="args" />`,
});

export const Default = Template.bind({});

Default.args = {
  isOpen: true,
  frame: normalizeSentryEvent(sentryEventMock)?.payload?.exception?.values?.[0]?.stacktrace?.frames[1],
};
