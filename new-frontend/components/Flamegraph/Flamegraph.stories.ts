import {Meta, Story} from "@storybook/vue3";
import {normalizeProfilerEvent} from "~/utils/normalize-event";
import profilerEventMock from '~/mocks/profiler.json'
import FlamegraphComponent from './Flamegraph.vue';

export default {
  title: "Profiler/Flamegraph",
  component: FlamegraphComponent
} as Meta<typeof FlamegraphComponent>;

const Template: Story = (args) => ({
  components: {FlamegraphComponent},
  setup() {
    return {
      args,
    };
  },
  template: `
    <FlamegraphComponent style="width: 100%; height: 100vh" v-bind="args"/>`,
});

export const Flamegraph = Template.bind({});

Flamegraph.args = {
  edges: normalizeProfilerEvent(profilerEventMock).payload.edges,
};
