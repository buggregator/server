import { Meta, Story } from "@storybook/vue3";
import { normalizeProfilerEvent } from "~/utils/normalize-event";
import profilerEventMock from '~/mocks/profiler.json'
import ProfilerPreview from '~/components/ProfilerPreview/ProfilerPreview.vue';

export default {
  title: "Profiler/Components/ProfilerPreview",
  component: ProfilerPreview
} as Meta<typeof ProfilerPreview>;

const Template: Story = (args) => ({
  components: { ProfilerPreview },
  setup() {
    return {
      args,
    };
  },
  template: `<ProfilerPreview v-bind="args" />`,
});

export const Event = Template.bind({});

Event.args = {
  event: normalizeProfilerEvent(profilerEventMock),
};
