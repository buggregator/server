import { Meta, Story } from "@storybook/vue3";
import { normalizeProfilerEvent } from "~/utils/normalize-event";
import PageProfiler from '~/components/PageProfiler/PageProfiler.vue';
import profilerEventMock from '~/mocks/profiler.json'

export default {
  title: "Pages/PageProfiler",
  component: PageProfiler
} as Meta<typeof PageProfiler>;

const Template: Story = (args) => ({
  components: { PageProfiler },
  setup() {
    return {
      args,
    };
  },
  template: `<page-profiler v-bind="args" />`,
});

export const Default = Template.bind({});

Default.args = {
  event: normalizeProfilerEvent(profilerEventMock),
};
