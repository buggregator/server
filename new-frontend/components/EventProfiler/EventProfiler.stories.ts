import { Meta, Story } from "@storybook/vue3";
import { normalizeProfilerEvent } from "~/utils/normalize-event";
import EventProfiler from '~/components/EventProfiler/EventProfiler.vue';
import profilerEventMock from '~/mocks/profiler.json'

export default {
  title: "Profiler",
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

export const Event = Template.bind({});

Event.args = {
  event: normalizeProfilerEvent(profilerEventMock),
};
