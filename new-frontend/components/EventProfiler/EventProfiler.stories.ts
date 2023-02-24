import { Meta, Story } from "@storybook/vue3";
import { normalizeSentryEvent } from "~/utils/normalizeEvent";
import EventProfiler from '~/components/EventProfiler/EventProfiler.vue';
import sentryEventMock from '~/mocks/sentry.json'

export default {
  title: "Event/EventProfiler",
  component: EventProfiler
} as Meta<typeof EventProfiler>;

const Template: Story = (args) => ({
  components: { EventProfiler },
  setup() {
    return {
      args,
    };
  },
  template: `<event-profiler v-bind="args" />`,
});

export const Default = Template.bind({});

Default.args = {
  event: normalizeSentryEvent(sentryEventMock),
};
