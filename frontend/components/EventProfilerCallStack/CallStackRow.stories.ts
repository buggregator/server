import { Meta, Story } from "@storybook/vue3";
import { normalizeProfilerEvent } from "~/utils/normalize-event";
import CallStackRow from '~/components/EventProfilerCallStack/CallStackRow.vue';
import profilerEventMock from '~/mocks/profiler.json'

export default {
  title: "Profiler/CallStack",
  component: CallStackRow
} as Meta<typeof CallStackRow>;

const Template: Story = (args) => ({
  components: { CallStackRow },
  setup() {
    return {
      args,
    };
  },
  template: `<call-stack-row v-bind="args" />`,
});

export const Row = Template.bind({});

Row.args = {
  edge: normalizeProfilerEvent(profilerEventMock).payload.edges.e5,
};
