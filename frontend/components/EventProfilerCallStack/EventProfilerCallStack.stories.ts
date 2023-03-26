import { Meta, Story } from "@storybook/vue3";
import { normalizeProfilerEvent } from "~/utils/normalize-event";
import CallStack from '~/components/EventProfilerCallStack/EventProfilerCallStack.vue';
import profilerEventMock from '~/mocks/profiler.json'

export default {
  title: "Profiler/CallStack",
  component: CallStack
} as Meta<typeof CallStack>;

const Template: Story = (args) => ({
  components: { CallStack },
  setup() {
    return {
      args,
    };
  },
  template: `<call-stack v-bind="args" />`,
});

export const List = Template.bind({});

List.args = {
  event: normalizeProfilerEvent(profilerEventMock).payload,
};
