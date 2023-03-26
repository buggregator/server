import {Meta, Story} from "@storybook/vue3";
import {normalizeInspectorEvent} from "~/utils/normalize-event";
import inspectorEventMock from '~/mocks/inspector.json'
import TimelineComponent from './Timeline.vue';

export default {
  title: "Pages/Inspector",
  component: TimelineComponent
} as Meta<typeof TimelineComponent>;

const Template: Story = (args) => ({
  components: {TimelineComponent},
  setup() {
    return {
      args,
    };
  },
  template: `<TimelineComponent v-bind="args"/>`,
});

export const Timeline = Template.bind({});

Timeline.args = {
  event: normalizeInspectorEvent(inspectorEventMock)
};
