import { Meta, Story } from "@storybook/vue3";
import { normalizeProfilerEvent } from "~/utils/normalizeEvent";
import StatBoard from '~/components/StatBoard/StatBoard.vue';
import profilerEventMock from '~/mocks/profiler.json'
import type { Profiler } from '~/config/types'

export default {
  title: "Components/StatBoard",
  component: StatBoard
} as Meta<typeof StatBoard>;

const Template: Story = (args) => ({
  components: { StatBoard },
  setup() {
    return {
      args,
    };
  },
  template: `<stat-board v-bind="args" />`,
});

export const Default = Template.bind({});

Default.args = {
  cost: (normalizeProfilerEvent(profilerEventMock).payload as Profiler)?.peaks,
};
