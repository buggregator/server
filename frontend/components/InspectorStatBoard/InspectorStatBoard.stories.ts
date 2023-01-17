import {Meta, Story} from "@storybook/vue3";
import {normalizeInspectorEvent} from "~/utils/normalize-event";
import inspectorEventMock from '~/mocks/inspector.json'
import InspectorStatBoard from '~/components/InspectorStatBoard/InspectorStatBoard.vue';

export default {
  title: "Inspector/Components/InspectorStatBoard",
  component: InspectorStatBoard
} as Meta<typeof InspectorStatBoard>;

const Template: Story = (args) => ({
  components: {InspectorStatBoard},
  setup() {
    return {
      args,
    };
  },
  template: `<InspectorStatBoard v-bind="args"/>`,
});

export const StatBoardStories = Template.bind({});

StatBoardStories.args = {
  transaction: normalizeInspectorEvent(inspectorEventMock).payload[0]
};
