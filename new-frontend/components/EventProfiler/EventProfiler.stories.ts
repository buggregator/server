import { Meta, Story } from "@storybook/vue3";
import { normalizeProfilerEvent } from "~/utils/normalize-event";
import EventProfiler from '~/components/EventProfiler/EventProfiler.vue';
import profilerEventMock from '~/mocks/profiler.json'

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
  event: normalizeProfilerEvent(profilerEventMock),
};
