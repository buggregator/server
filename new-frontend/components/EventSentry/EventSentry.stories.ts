import { Meta, Story } from "@storybook/vue3";
import { normalizeSentryEvent } from "~/utils/normalize-event";
import EventSentry from '~/components/EventSentry/EventSentry.vue';
import sentryEventMock from '~/mocks/sentry.json'

export default {
  title: "Event/EventSentry",
  component: EventSentry
} as Meta<typeof EventSentry>;

const Template: Story = (args) => ({
  components: { EventSentry },
  setup() {
    return {
      args,
    };
  },
  template: `<event-sentry v-bind="args" />`,
});

export const Default = Template.bind({});

Default.args = {
  event: normalizeSentryEvent(sentryEventMock),
};
