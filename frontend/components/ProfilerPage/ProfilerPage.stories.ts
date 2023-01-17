import { Meta, Story } from "@storybook/vue3";
import { normalizeProfilerEvent } from "~/utils/normalize-event";
import ProfilerPage from '~/components/ProfilerPage/ProfilerPage.vue';
import profilerEventMock from '~/mocks/profiler.json'

export default {
  title: "Profiler/Page/ProfilerPage",
  component: ProfilerPage
} as Meta<typeof ProfilerPage>;

const Template: Story = (args) => ({
  components: { ProfilerPage },
  setup() {
    return {
      args,
    };
  },
  template: `<ProfilerPage v-bind="args" />`,
});

export const Default = Template.bind({});

Default.args = {
  event: normalizeProfilerEvent(profilerEventMock),
};
