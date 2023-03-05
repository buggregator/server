import { Meta, Story } from "@storybook/vue3";
import { normalizeSentryEvent } from "~/utils/normalize-event";
import EventSentryFrame, { SentryFrame } from '~/components/EventSentryFrame/EventSentryFrame.vue';
import sentryEventMock from '~/mocks/sentry.json'
import { Sentry } from "~/config/types";

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
  frame: (normalizeSentryEvent(sentryEventMock)?.payload as Sentry)?.exception?.values[0]?.stacktrace?.frames[1] as SentryFrame,
};
