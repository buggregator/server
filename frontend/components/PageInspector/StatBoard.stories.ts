import {Meta, Story} from "@storybook/vue3";
import {normalizeInspectorEvent} from "~/utils/normalize-event";
import inspectorEventMock from '~/mocks/inspector.json'
import StatBoardComponent from './StatBoard.vue';

export default {
  title: "Pages/Inspector",
  component: StatBoardComponent
} as Meta<typeof StatBoardComponent>;

const Template: Story = (args) => ({
  components: {StatBoardComponent},
  setup() {
    return {
      args,
    };
  },
  template: `<StatBoardComponent v-bind="args"/>`,
});

export const StatBoardStories = Template.bind({});

StatBoardStories.args = {
  transaction: normalizeInspectorEvent(inspectorEventMock).payload[0]
};
