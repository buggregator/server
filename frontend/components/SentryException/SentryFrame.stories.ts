import { Meta, Story } from "@storybook/vue3";
import { normalizeSentryEvent } from "~/utils/normalize-event";
import sentrySpiralEventMock from '~/mocks/sentry-spiral.json'
import { Sentry } from "~/config/types";
import EventSentryFrame, { SentryFrame } from './SentryFrame.vue';

export default {
  title: "Sentry/Components",
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

export const Frame = Template.bind({});

Frame.args = {
  isOpen: true,
  frame: (normalizeSentryEvent(sentrySpiralEventMock)?.payload as Sentry)?.exception?.values[0]?.stacktrace?.frames[1] as SentryFrame,
};