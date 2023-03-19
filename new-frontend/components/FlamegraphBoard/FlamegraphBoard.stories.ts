import {Meta, Story} from "@storybook/vue3";
import {normalizeProfilerEvent} from "~/utils/normalize-event";
import profilerEventMock from '~/mocks/profiler.json'
import { Profiler } from "~/config/types";
import FlamegraphBoard from './FlamegraphBoard.vue';

export default {
  title: "Profiler/FlamegraphBoard",
  component: FlamegraphBoard
} as Meta<typeof FlamegraphBoard>;

const Template: Story = (args) => ({
  components: {FlamegraphBoard},
  setup() {
    return {
      args,
    };
  },
  template: `<flamegraph-board style="width: 100%; height: 100vh" v-bind="args"/>`,
});

export const Default = Template.bind({});

Default.args = {
  edges: (normalizeProfilerEvent(profilerEventMock).payload as Profiler).edges,
};
