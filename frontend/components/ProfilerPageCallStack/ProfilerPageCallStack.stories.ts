import { Meta, Story } from "@storybook/vue3";
import { normalizeProfilerEvent } from "~/utils/normalize-event";
import ProfilerPageCallStack from '~/components/ProfilerPageCallStack/ProfilerPageCallStack.vue';
import profilerEventMock from '~/mocks/profiler.json'

export default {
  title: "Profiler/Page/ProfilerPageCallStack",
  component: ProfilerPageCallStack
} as Meta<typeof ProfilerPageCallStack>;

const Template: Story = (args) => ({
  components: { ProfilerPageCallStack },
  setup() {
    return {
      args,
    };
  },
  template: `<ProfilerPageCallStack v-bind="args" />`,
});

export const List = Template.bind({});

List.args = {
  event: normalizeProfilerEvent(profilerEventMock).payload,
};
