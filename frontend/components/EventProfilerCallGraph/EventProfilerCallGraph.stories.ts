import { Meta, Story } from "@storybook/vue3";
import { normalizeProfilerEvent } from "~/utils/normalize-event";
import CallGraphEvent from '~/components/EventProfilerCallGraph/EventProfilerCallGraph.vue';
import profilerEventMock from '~/mocks/profiler.json'

export default {
  title: "Profiler",
  component: CallGraphEvent
} as Meta<typeof CallGraphEvent>;

const Template: Story = (args) => ({
  components: { CallGraphEvent },
  setup() {
    return {
      args,
    };
  },
  template: `<call-graph-event v-bind="args" />`,
});

export const CallGraph = Template.bind({});

CallGraph.args = {
  event: normalizeProfilerEvent(profilerEventMock).payload,
};
