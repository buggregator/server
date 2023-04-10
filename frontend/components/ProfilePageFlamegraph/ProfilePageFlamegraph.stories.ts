import {Meta, Story} from "@storybook/vue3";
import {normalizeProfilerEvent} from "~/utils/normalize-event";
import profilerEventMock from '~/mocks/profiler.json'
import { Profiler } from "~/config/types";
import ProfilePageFlamegraph from '~/components/ProfilePageFlamegraph/ProfilePageFlamegraph.vue';

export default {
  title: "Profiler/Page/ProfilePageFlamegraph",
  component: ProfilePageFlamegraph
} as Meta<typeof ProfilePageFlamegraph>;

const Template: Story = (args) => ({
  components: {ProfilePageFlamegraph},
  setup() {
    return {
      args,
    };
  },
  template: `<ProfilePageFlamegraph style="width: 100%; height: 100vh" v-bind="args"/>`,
});

export const Default = Template.bind({});

Default.args = {
  edges: (normalizeProfilerEvent(profilerEventMock).payload as Profiler).edges,
};
